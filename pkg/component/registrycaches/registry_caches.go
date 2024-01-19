// Copyright (c) 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registrycaches

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/utils"
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/helper"
	"github.com/gardener/gardener-extension-registry-cache/pkg/constants"
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

// Values is a set of configuration values for the registry caches.
type Values struct {
	// Image is the container image used for the registry cache.
	Image string
	// VPAEnabled marks whether VerticalPodAutoscaler is enabled for the shoot.
	VPAEnabled bool
	// PSPDisabled marks whether the PodSecurityPolicy admission plugin is disabled.
	PSPDisabled bool
	// Caches are the registry caches to deploy.
	Caches []api.RegistryCache
	// ResourceReferences are the resource references from the Shoot spec (the .spec.resources field).
	ResourceReferences []gardencorev1beta1.NamedResourceReference
	// KeepObjectsOnDestroy marks whether the ManagedResource's .spec.keepObjects will be set to true
	// before ManagedResource deletion during the Destroy operation. When set to true, the deployed
	// resources by ManagedResources won't be deleted, but the ManagedResource itself will be deleted.
	KeepObjectsOnDestroy bool
}

// New creates a new instance of DeployWaiter for registry caches.
func New(
	client client.Client,
	namespace string,
	values Values,
) component.DeployWaiter {
	return &registryCaches{
		client:    client,
		namespace: namespace,
		values:    values,
	}
}

type registryCaches struct {
	client    client.Client
	namespace string
	values    Values
}

// Deploy implements component.DeployWaiter.
func (r *registryCaches) Deploy(ctx context.Context) error {
	data, err := r.computeResourcesData(ctx)
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
		return fmt.Errorf("failed to not create or update managed resource: %w", err)
	}

	if err := r.deployMonitoringConfigMap(ctx); err != nil {
		return fmt.Errorf("failed to deploy monitoring ConfigMap: %w", err)
	}

	return nil
}

// Destroy implements component.DeployWaiter.
func (r *registryCaches) Destroy(ctx context.Context) error {
	if r.values.KeepObjectsOnDestroy {
		if err := managedresources.SetKeepObjects(ctx, r.client, r.namespace, managedResourceName, true); err != nil {
			return err
		}
	}

	return managedresources.Delete(ctx, r.client, r.namespace, managedResourceName, false)
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

func (r *registryCaches) computeResourcesData(ctx context.Context) (map[string][]byte, error) {
	var objects []client.Object

	serviceAccountName := "default"
	if !r.values.PSPDisabled {
		serviceAccountName = "registry-cache"

		serviceAccount := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceAccountName,
				Namespace: metav1.NamespaceSystem,
			},
			AutomountServiceAccountToken: pointer.Bool(false),
		}
		podSecurityPolicy := &policyv1beta1.PodSecurityPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.kube-system.registry-cache",
				Annotations: map[string]string{
					v1beta1constants.AnnotationSeccompDefaultProfile:  v1beta1constants.AnnotationSeccompAllowedProfilesRuntimeDefaultValue,
					v1beta1constants.AnnotationSeccompAllowedProfiles: v1beta1constants.AnnotationSeccompAllowedProfilesRuntimeDefaultValue,
				},
			},
			Spec: policyv1beta1.PodSecurityPolicySpec{
				RunAsUser: policyv1beta1.RunAsUserStrategyOptions{
					Rule: policyv1beta1.RunAsUserStrategyRunAsAny,
				},
				SELinux: policyv1beta1.SELinuxStrategyOptions{
					Rule: policyv1beta1.SELinuxStrategyRunAsAny,
				},
				SupplementalGroups: policyv1beta1.SupplementalGroupsStrategyOptions{
					Rule: policyv1beta1.SupplementalGroupsStrategyRunAsAny,
				},
				FSGroup: policyv1beta1.FSGroupStrategyOptions{
					Rule: policyv1beta1.FSGroupStrategyRunAsAny,
				},
				Volumes: []policyv1beta1.FSType{
					policyv1beta1.PersistentVolumeClaim,
				},
			},
		}
		clusterRolePSP := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:psp:kube-system:registry-cache",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{"policy", "extensions"},
					ResourceNames: []string{podSecurityPolicy.Name},
					Resources:     []string{"podsecuritypolicies"},
					Verbs:         []string{"use"},
				},
			},
		}
		roleBindingPSP := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gardener.cloud:psp:registry-cache",
				Namespace: metav1.NamespaceSystem,
				Annotations: map[string]string{
					resourcesv1alpha1.DeleteOnInvalidUpdate: "true",
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     clusterRolePSP.Name,
			},
			Subjects: []rbacv1.Subject{{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      serviceAccount.Name,
				Namespace: serviceAccount.Namespace,
			}},
		}

		objects = append(objects,
			serviceAccount,
			podSecurityPolicy,
			clusterRolePSP,
			roleBindingPSP,
		)
	}

	for _, cache := range r.values.Caches {
		cacheObjects, err := r.computeResourcesDataForRegistryCache(ctx, &cache, serviceAccountName)
		if err != nil {
			return nil, fmt.Errorf("failed to compute resources for upstream %s: %w", cache.Upstream, err)
		}

		objects = append(objects, cacheObjects...)
	}

	registry := managedresources.NewRegistry(kubernetes.ShootScheme, kubernetes.ShootCodec, kubernetes.ShootSerializer)

	return registry.AddAllAndSerialize(objects...)
}

func (r *registryCaches) computeResourcesDataForRegistryCache(ctx context.Context, cache *api.RegistryCache, serviceAccountName string) ([]client.Object, error) {
	if cache.Volume == nil || cache.Volume.Size == nil {
		return nil, fmt.Errorf("registry cache volume size is required")
	}

	const (
		registryCacheVolumeName  = "cache-volume"
		registryConfigVolumeName = "config-volume"
		debugPort                = 5001
	)

	var (
		name         = computeName(cache.Upstream)
		configValues = map[string]interface{}{
			"http_addr":              fmt.Sprintf(":%d", constants.RegistryCachePort),
			"http_debug_addr":        fmt.Sprintf(":%d", debugPort),
			"proxy_remoteurl":        registryutils.GetUpstreamURL(cache.Upstream),
			"storage_delete_enabled": strconv.FormatBool(helper.GarbageCollectionEnabled(cache)),
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
			Labels:    getLabels(name, cache.Upstream),
		},
		Data: map[string][]byte{
			"config.yml": configYAML.Bytes(),
		},
	}
	utilruntime.Must(kubernetesutils.MakeUnique(configSecret))

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceSystem,
			Labels:    getLabels(name, cache.Upstream),
		},
		Spec: corev1.ServiceSpec{
			Selector: getLabels(name, cache.Upstream),
			Ports: []corev1.ServicePort{{
				Name:       "registry-cache",
				Port:       constants.RegistryCachePort,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromString("registry-cache"),
			}},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceSystem,
			Labels:    getLabels(name, cache.Upstream),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: service.Name,
			Selector: &metav1.LabelSelector{
				MatchLabels: getLabels(name, cache.Upstream),
			},
			Replicas: pointer.Int32(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: utils.MergeStringMaps(getLabels(name, cache.Upstream), map[string]string{
						v1beta1constants.LabelNetworkPolicyToDNS:            v1beta1constants.LabelNetworkPolicyAllowed,
						v1beta1constants.LabelNetworkPolicyToPublicNetworks: v1beta1constants.LabelNetworkPolicyAllowed,
					}),
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: pointer.Bool(false),
					ServiceAccountName:           serviceAccountName,
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
									MountPath: "/var/lib/registry",
								},
								{
									Name:      registryConfigVolumeName,
									MountPath: "/etc/docker/registry",
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
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   registryCacheVolumeName,
						Labels: getLabels(name, cache.Upstream),
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
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

	var vpa *vpaautoscalingv1.VerticalPodAutoscaler
	if r.values.VPAEnabled {
		updateMode := vpaautoscalingv1.UpdateModeAuto
		controlledValues := vpaautoscalingv1.ContainerControlledValuesRequestsOnly
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
					UpdateMode: &updateMode,
				},
				ResourcePolicy: &vpaautoscalingv1.PodResourcePolicy{
					ContainerPolicies: []vpaautoscalingv1.ContainerResourcePolicy{
						{
							ContainerName:    vpaautoscalingv1.DefaultContainerResourcePolicy,
							ControlledValues: &controlledValues,
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
		service,
		statefulSet,
		vpa,
	}, nil
}

func getLabels(name, upstream string) map[string]string {
	return map[string]string{
		"app":                       name,
		constants.UpstreamHostLabel: upstream,
	}
}

// computeName computes a name by given upstream.
// The name later on is used by the registry cache Service, config Secret, StatefulSet and VPA.
//
// If length of registry-<escaped_upsteam> is NOT > 52, the name is registry-<escaped_upsteam>.
// Otherwise it is registry-<truncated_escaped_upsteam>-<hash> where <escaped_upsteam> is truncated at 37 chars.
// The returned name is at most 52 chars.
func computeName(upstream string) string {
	// The StatefulSet name limit is 63 chars. However Pods for a StatefulSet with name > 52 chars cannot be created due to https://github.com/kubernetes/kubernetes/issues/64023.
	// The "controller-revision-hash" label gets added to the StatefulSet Pod. The label value is in format <stateful_set_name>_<hash> where <hash> is 10 or 11 chars.
	// A label value limit is 63 chars. That's why a Pod for a StatefulSet with name > 52 chars cannot be created.
	const statefulSetNameLimit = 52

	escapedUpstream := strings.Replace(upstream, ".", "-", -1)
	name := "registry-" + escapedUpstream
	if len(name) > statefulSetNameLimit {
		hash := utils.ComputeSHA256Hex([]byte(upstream))[:5]
		upstreamLimit := statefulSetNameLimit - len("registry-") - len(hash) - 1
		name = fmt.Sprintf("registry-%s-%s", escapedUpstream[:upstreamLimit], hash)
	}

	return name
}
