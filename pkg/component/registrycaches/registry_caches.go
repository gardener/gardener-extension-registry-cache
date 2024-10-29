// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registrycaches

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionssecretsmanager "github.com/gardener/gardener/extensions/pkg/util/secret/manager"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/utils"
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/helper"
	"github.com/gardener/gardener-extension-registry-cache/pkg/constants"
	"github.com/gardener/gardener-extension-registry-cache/pkg/secrets"
	registryutils "github.com/gardener/gardener-extension-registry-cache/pkg/utils/registry"
)

const (
	managedResourceName = "extension-registry-cache"
)

var (
	//go:embed templates/config.yml.tpl
	configContentTpl string
	configTpl        *template.Template
)

func init() {
	var err error
	configTpl, err = template.
		New("config.yml.tpl").
		Parse(configContentTpl)
	utilruntime.Must(err)
}

// Interface is an interface for managing Registry Caches.
type Interface interface {
	component.DeployWaiter
	// CASecretName returns the name of the CA secret.
	CASecretName() string
}

// Values is a set of configuration values for the registry caches.
type Values struct {
	// Image is the container image used for the registry cache.
	Image string
	// VPAEnabled marks whether VerticalPodAutoscaler is enabled for the shoot.
	VPAEnabled bool
	// Services are the registry cache services used for certificate generation.
	Services []corev1.Service
	// Caches are the registry caches to deploy.
	Caches []api.RegistryCache
	// ResourceReferences are the resource references from the Shoot spec (the .spec.resources field).
	ResourceReferences []gardencorev1beta1.NamedResourceReference
	// KeepObjectsOnDestroy marks whether the ManagedResource's .spec.keepObjects will be set to true
	// before ManagedResource deletion during the Destroy operation. When set to true, the deployed
	// resources by ManagedResources won't be deleted, but the ManagedResource itself will be deleted.
	KeepObjectsOnDestroy bool
}

// NewComponent creates a new instance of Interface for registry caches.
func NewComponent(
	client client.Client,
	secretManager secretsmanager.Interface,
	namespace string,
	values Values,
) Interface {
	return &registryCaches{
		client:        client,
		secretManager: secretManager,
		namespace:     namespace,
		values:        values,
	}
}

type registryCaches struct {
	client        client.Client
	secretManager secretsmanager.Interface
	namespace     string
	values        Values

	caSecretName string
}

// Deploy implements component.DeployWaiter.
func (r *registryCaches) Deploy(ctx context.Context) error {
	secretsConfig := secrets.ConfigsFor(r.values.Services)
	generatedSecrets, err := extensionssecretsmanager.GenerateAllSecrets(ctx, r.secretManager, secretsConfig)
	if err != nil {
		return err
	}

	caSecret, found := r.secretManager.Get(secrets.CAName)
	if !found {
		return fmt.Errorf("secret %q not found", secrets.CAName)
	}
	r.caSecretName = caSecret.Name

	data, err := r.computeResourcesData(ctx, generatedSecrets)
	if err != nil {
		return err
	}

	var (
		keepObjects = false

		secretName, secret = managedresources.NewSecret(r.client, r.namespace, managedResourceName, data, false)
		managedResource    = managedresources.NewForShoot(r.client, r.namespace, managedResourceName, constants.Origin, keepObjects).
					WithSecretRef(secretName).
					DeletePersistentVolumeClaims(true)
	)

	if err := secret.Reconcile(ctx); err != nil {
		return fmt.Errorf("failed to create or update secret of managed resources: %w", err)
	}
	if err := managedResource.Reconcile(ctx); err != nil {
		return fmt.Errorf("failed to create or update managed resource: %w", err)
	}

	if err := r.deployMonitoringConfig(ctx); err != nil {
		return fmt.Errorf("failed to deploy monitoring config: %w", err)
	}

	return nil
}

// Destroy implements component.DeployWaiter.
func (r *registryCaches) Destroy(ctx context.Context) error {
	if r.values.KeepObjectsOnDestroy {
		if err := managedresources.SetKeepObjects(ctx, r.client, r.namespace, managedResourceName, true); err != nil {
			return fmt.Errorf("failed to set keep objects to managed resource: %w", err)
		}
	}

	if err := managedresources.Delete(ctx, r.client, r.namespace, managedResourceName, false); err != nil {
		return fmt.Errorf("failed to delete managed resource: %w", err)
	}

	if err := r.destroyMonitoringConfig(ctx); err != nil {
		return fmt.Errorf("failed to destroy monitoring config: %w", err)
	}

	return nil
}

// TimeoutWaitForManagedResource is the timeout used while waiting for the ManagedResources to become healthy
// or deleted.
var TimeoutWaitForManagedResource = 2 * time.Minute

// Wait implements component.DeployWaiter.
func (r *registryCaches) Wait(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitForManagedResource)
	defer cancel()

	return managedresources.WaitUntilHealthy(timeoutCtx, r.client, r.namespace, managedResourceName)
}

// WaitCleanup implements component.DeployWaiter.
func (r *registryCaches) WaitCleanup(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitForManagedResource)
	defer cancel()

	return managedresources.WaitUntilDeleted(timeoutCtx, r.client, r.namespace, managedResourceName)
}

func (r *registryCaches) CASecretName() string {
	return r.caSecretName
}

func (r *registryCaches) computeResourcesData(ctx context.Context, secrets map[string]*corev1.Secret) (map[string][]byte, error) {
	var objects []client.Object

	remappedSecrets := make(map[string]*corev1.Secret, len(secrets))
	for _, secret := range secrets {
		remappedSecrets[strings.TrimSuffix(secret.Labels["name"], "-tls")] = secret
	}

	for _, cache := range r.values.Caches {
		secret, ok := remappedSecrets[cache.Upstream]
		if !ok {
			return nil, fmt.Errorf("secret for %s upstream not found", cache.Upstream)
		}
		cacheObjects, err := r.computeResourcesDataForRegistryCache(ctx, &cache, secret)
		if err != nil {
			return nil, fmt.Errorf("failed to compute resources for upstream %s: %w", cache.Upstream, err)
		}

		objects = append(objects, cacheObjects...)
	}

	registry := managedresources.NewRegistry(kubernetes.ShootScheme, kubernetes.ShootCodec, kubernetes.ShootSerializer)

	return registry.AddAllAndSerialize(objects...)
}

func (r *registryCaches) computeResourcesDataForRegistryCache(ctx context.Context, cache *api.RegistryCache, secret *corev1.Secret) ([]client.Object, error) {
	if cache.Volume == nil || cache.Volume.Size == nil {
		return nil, fmt.Errorf("registry cache volume size is required")
	}

	const (
		checksumAnnotation       = "checksum/secret-%s-tls"
		registryCacheVolumeName  = "cache-volume"
		registryConfigVolumeName = "config-volume"
		registryCertsVolumeName  = "certs-volume"
		repositoryMountPath      = "/var/lib/registry"
		debugPort                = 5001
	)

	var (
		upstreamLabel = registryutils.ComputeUpstreamLabelValue(cache.Upstream)
		name          = "registry-" + strings.ReplaceAll(upstreamLabel, ".", "-")
		remoteURL     = ptr.Deref(cache.RemoteURL, registryutils.GetUpstreamURL(cache.Upstream))
		configValues  = map[string]interface{}{
			"http_addr":       fmt.Sprintf(":%d", constants.RegistryCachePort),
			"http_debug_addr": fmt.Sprintf(":%d", debugPort),
			"proxy_remoteurl": remoteURL,
			"proxy_ttl":       helper.GarbageCollectionTTL(cache).Duration.String(),
		}
	)

	var storageClassName *string
	if cache.Volume != nil {
		storageClassName = cache.Volume.StorageClassName
	}

	if cache.SecretReferenceName != nil {
		ref := v1beta1helper.GetResourceByName(r.values.ResourceReferences, *cache.SecretReferenceName)
		if ref == nil || ref.ResourceRef.Kind != "Secret" {
			return nil, fmt.Errorf("failed to find referenced resource with name %s and kind Secret", *cache.SecretReferenceName)
		}

		refSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ref.ResourceRef.Name,
				Namespace: r.namespace,
			},
		}
		if err := controller.GetObjectByReference(ctx, r.client, &ref.ResourceRef, r.namespace, refSecret); err != nil {
			return nil, fmt.Errorf("failed to read referenced secret %s%s for reference %s", v1beta1constants.ReferencedResourcesPrefix, ref.ResourceRef.Name, *cache.SecretReferenceName)
		}

		configValues["proxy_username"] = string(refSecret.Data["username"])
		configValues["proxy_password"] = string(refSecret.Data["password"])
	}

	var configYAML bytes.Buffer
	if err := configTpl.Execute(&configYAML, configValues); err != nil {
		return nil, err
	}

	configSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-config",
			Namespace: metav1.NamespaceSystem,
			Labels:    registryutils.GetLabels(name, upstreamLabel),
		},
		Data: map[string][]byte{
			"config.yml": configYAML.Bytes(),
		},
	}
	utilruntime.Must(kubernetesutils.MakeUnique(configSecret))

	secretTLS := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-tls",
			Namespace: metav1.NamespaceSystem,
			Labels:    registryutils.GetLabels(name, upstreamLabel),
		},
		Type: corev1.SecretTypeOpaque,
		Data: secret.Data,
	}
	utilruntime.Must(kubernetesutils.MakeUnique(secretTLS))

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceSystem,
			Labels:    registryutils.GetLabels(name, upstreamLabel),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: name,
			Selector: &metav1.LabelSelector{
				MatchLabels: registryutils.GetLabels(name, upstreamLabel),
			},
			Replicas: ptr.To[int32](1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: utils.MergeStringMaps(registryutils.GetLabels(name, upstreamLabel), map[string]string{
						v1beta1constants.LabelNetworkPolicyToDNS:            v1beta1constants.LabelNetworkPolicyAllowed,
						v1beta1constants.LabelNetworkPolicyToPublicNetworks: v1beta1constants.LabelNetworkPolicyAllowed,
					}),
					Annotations: map[string]string{
						fmt.Sprintf(checksumAnnotation, name): utils.ComputeChecksum(secretTLS.Data),
					},
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: ptr.To(false),
					PriorityClassName:            "system-cluster-critical",
					SecurityContext: &corev1.PodSecurityContext{
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "registry-cache",
							Image:           r.values.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("20m"),
									corev1.ResourceMemory: resource.MustParse("50Mi"),
								},
							},
							// Mitigation for https://github.com/distribution/distribution/issues/4478:
							// The registry image entrypoint (https://github.com/distribution/distribution-library-image/blob/be4eca0a5f3af34a026d1e9294d63f3464c06131/Dockerfile#L31)
							// is extended with a mitigation logic for https://github.com/distribution/distribution/issues/4478.
							// Keep in sync the registry image entrypoint with the below invocation when updating the registry image version.
							Command: []string{"/bin/sh", "-c", `REPO_ROOT=` + repositoryMountPath + `
SCHEDULER_STATE_FILE="${REPO_ROOT}/scheduler-state.json"

if [ -f "${SCHEDULER_STATE_FILE}" ]; then
    if [ -s "${SCHEDULER_STATE_FILE}" ]; then
        echo "The scheduler-state.json file exists and it is not empty. Won't clean up anything..."
    else
        echo "Detected a corrupted scheduler-state.json file"

        echo "Cleaning up the scheduler-state.json file"
        rm -f "${SCHEDULER_STATE_FILE}"

        echo "Cleaning up the docker directory"
        rm -rf "${REPO_ROOT}/docker"
    fi
else
    echo "The scheduler-state.json file is not created yet. Won't clean up anything..."
fi

echo "Starting..."
source /entrypoint.sh /etc/distribution/config.yml
`},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: constants.RegistryCachePort,
									Name:          "registry-cache",
								},
								{
									ContainerPort: debugPort,
									Name:          "debug",
								},
							},
							Env: []corev1.EnvVar{
								// Mitigation for https://github.com/distribution/distribution/issues/4270.
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "none",
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/debug/health",
										Port: intstr.FromInt32(debugPort),
									},
								},
								FailureThreshold: 6,
								SuccessThreshold: 1,
								PeriodSeconds:    20,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/debug/health",
										Port: intstr.FromInt32(debugPort),
									},
								},
								FailureThreshold: 3,
								SuccessThreshold: 1,
								PeriodSeconds:    20,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      registryCacheVolumeName,
									ReadOnly:  false,
									MountPath: repositoryMountPath,
								},
								{
									Name:      registryConfigVolumeName,
									MountPath: "/etc/distribution",
								},
								{
									Name:      registryCertsVolumeName,
									MountPath: "/etc/distribution/certs",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: registryConfigVolumeName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: configSecret.Name,
								},
							},
						},
						{
							Name: registryCertsVolumeName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: secretTLS.Name,
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   registryCacheVolumeName,
						Labels: registryutils.GetLabels(name, upstreamLabel),
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: *cache.Volume.Size,
							},
						},
						StorageClassName: storageClassName,
					},
				},
			},
		},
	}

	if cache.Proxy != nil {
		if cache.Proxy.HTTPProxy != nil {
			statefulSet.Spec.Template.Spec.Containers[0].Env = append(statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
				Name:  "HTTP_PROXY",
				Value: *cache.Proxy.HTTPProxy,
			})
		}
		if cache.Proxy.HTTPSProxy != nil {
			statefulSet.Spec.Template.Spec.Containers[0].Env = append(statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
				Name:  "HTTPS_PROXY",
				Value: *cache.Proxy.HTTPSProxy,
			})
		}
	}

	var vpa *vpaautoscalingv1.VerticalPodAutoscaler
	if r.values.VPAEnabled {
		vpa = &vpaautoscalingv1.VerticalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: metav1.NamespaceSystem,
			},
			Spec: vpaautoscalingv1.VerticalPodAutoscalerSpec{
				TargetRef: &autoscalingv1.CrossVersionObjectReference{
					APIVersion: appsv1.SchemeGroupVersion.String(),
					Kind:       "StatefulSet",
					Name:       name,
				},
				UpdatePolicy: &vpaautoscalingv1.PodUpdatePolicy{
					UpdateMode: ptr.To(vpaautoscalingv1.UpdateModeAuto),
				},
				ResourcePolicy: &vpaautoscalingv1.PodResourcePolicy{
					ContainerPolicies: []vpaautoscalingv1.ContainerResourcePolicy{
						{
							ContainerName:    vpaautoscalingv1.DefaultContainerResourcePolicy,
							ControlledValues: ptr.To(vpaautoscalingv1.ContainerControlledValuesRequestsOnly),
							MinAllowed: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("20Mi"),
							},
							MaxAllowed: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("4"),
								corev1.ResourceMemory: resource.MustParse("8Gi"),
							},
						},
					},
				},
			},
		}
	}

	return []client.Object{
		configSecret,
		secretTLS,
		statefulSet,
		vpa,
	}, nil
}
