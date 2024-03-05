// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registryconfigurationcleaner

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-registry-cache/pkg/constants"
)

const (
	managedResourceName = "extension-registry-configuration-cleaner"
)

// Values is a set of configuration values for the registry configuration cleaner.
type Values struct {
	// AlpineImage is the alpine container image.
	AlpineImage string
	// PauseImage is the pause container image.
	PauseImage string
	// DeleteSystemdUnit represents whether the cleaner should delete the configure-containerd-registries.service systemd unit.
	DeleteSystemdUnit bool
	// Upstreams are the upstreams which registry configuration will be removed by the cleaner.
	Upstreams []string
}

// New creates a new instance of DeployWaiter for registry configuration cleaner.
func New(
	client client.Client,
	namespace string,
	values Values,
) component.DeployWaiter {
	return &registryConfigurationCleaner{
		client:    client,
		namespace: namespace,
		values:    values,
	}
}

type registryConfigurationCleaner struct {
	client    client.Client
	namespace string
	values    Values
}

// Deploy implements component.DeployWaiter.
func (r *registryConfigurationCleaner) Deploy(ctx context.Context) error {
	data, err := r.computeResourcesData()
	if err != nil {
		return err
	}

	return managedresources.CreateForShoot(ctx, r.client, r.namespace, managedResourceName, constants.Origin, false, data)
}

// Destroy implements component.DeployWaiter.
func (r *registryConfigurationCleaner) Destroy(ctx context.Context) error {
	return managedresources.Delete(ctx, r.client, r.namespace, managedResourceName, false)
}

// TimeoutWaitForManagedResource is the timeout used while waiting for the ManagedResources to become healthy.
var TimeoutWaitForManagedResource = 3 * time.Minute

// Wait implements component.DeployWaiter.
func (r *registryConfigurationCleaner) Wait(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitForManagedResource)
	defer cancel()

	return managedresources.WaitUntilHealthy(timeoutCtx, r.client, r.namespace, managedResourceName)
}

// TimeoutWaitCleanupForManagedResource is the timeout used while waiting for the ManagedResource to be deleted.
var TimeoutWaitCleanupForManagedResource = 2 * time.Minute

// WaitCleanup implements component.DeployWaiter.
func (r *registryConfigurationCleaner) WaitCleanup(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitCleanupForManagedResource)
	defer cancel()

	return managedresources.WaitUntilDeleted(timeoutCtx, r.client, r.namespace, managedResourceName)
}

func (r *registryConfigurationCleaner) computeResourcesData() (map[string][]byte, error) {
	if len(r.values.Upstreams) == 0 {
		return nil, fmt.Errorf("upstreams are required")
	}

	var (
		registry = managedresources.NewRegistry(kubernetes.ShootScheme, kubernetes.ShootCodec, kubernetes.ShootSerializer)

		daemonSet = &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "registry-configuration-cleaner",
				Namespace: metav1.NamespaceSystem,
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: getLabels(),
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: getLabels(),
					},
					Spec: corev1.PodSpec{
						AutomountServiceAccountToken: ptr.To(false),
						PriorityClassName:            v1beta1constants.PriorityClassNameShootSystem700,
						SecurityContext: &corev1.PodSecurityContext{
							SeccompProfile: &corev1.SeccompProfile{
								Type: corev1.SeccompProfileTypeRuntimeDefault,
							},
						},
						InitContainers: []corev1.Container{
							{
								Name:            "registry-configuration-cleaner",
								Image:           r.values.AlpineImage,
								ImagePullPolicy: corev1.PullIfNotPresent,
								Command:         computeCommand(r.values.DeleteSystemdUnit, r.values.Upstreams),
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("10m"),
										corev1.ResourceMemory: resource.MustParse("8Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:             "host-root-volume",
										MountPath:        "/host",
										ReadOnly:         false,
										MountPropagation: ptr.To(corev1.MountPropagationHostToContainer),
									},
								},
							},
						},
						Containers: []corev1.Container{
							{
								Name:            "pause",
								Image:           r.values.PauseImage,
								ImagePullPolicy: corev1.PullIfNotPresent,
							},
						},
						HostPID: true,
						Volumes: []corev1.Volume{
							{
								Name: "host-root-volume",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/",
									},
								},
							},
						},
					},
				},
			},
		}
	)

	return registry.AddAllAndSerialize(daemonSet)
}

func computeCommand(deleteSystemdUnit bool, upstreams []string) []string {
	command := []string{"sh", "-c"}

	var script strings.Builder

	if deleteSystemdUnit {
		script.WriteString(`if [[ -f /host/etc/systemd/system/configure-containerd-registries.service ]]; then
  chroot /host /bin/bash -c 'systemctl disable configure-containerd-registries.service; systemctl stop configure-containerd-registries.service; rm -f /etc/systemd/system/configure-containerd-registries.service'
fi
`)
		script.WriteString("\n")
	}

	for _, upstream := range upstreams {
		script.WriteString(fmt.Sprintf(`if [[ -d /host/etc/containerd/certs.d/%[1]s ]]; then
  rm -rf /host/etc/containerd/certs.d/%[1]s
fi
`, upstream))
	}

	command = append(command, script.String())

	return command
}

func getLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     "registry-configuration-cleaner",
		"app.kubernetes.io/instance": "registry-configuration-cleaner",
	}
}
