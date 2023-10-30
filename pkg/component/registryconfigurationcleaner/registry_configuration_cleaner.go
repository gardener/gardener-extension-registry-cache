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

package registryconfigurationcleaner

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
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
	// PSPDisabled marks whether the PodSecurityPolicy admission plugin is disabled.
	PSPDisabled bool
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

	var objects []client.Object

	serviceAccountName := "default"
	if !r.values.PSPDisabled {
		serviceAccountName = "registry-configuration-cleaner"

		serviceAccount := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceAccountName,
				Namespace: metav1.NamespaceSystem,
			},
			AutomountServiceAccountToken: pointer.Bool(false),
		}
		podSecurityPolicy := &policyv1beta1.PodSecurityPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.kube-system.registry-configuration-cleaner",
				Annotations: map[string]string{
					v1beta1constants.AnnotationSeccompDefaultProfile:  v1beta1constants.AnnotationSeccompAllowedProfilesRuntimeDefaultValue,
					v1beta1constants.AnnotationSeccompAllowedProfiles: v1beta1constants.AnnotationSeccompAllowedProfilesRuntimeDefaultValue,
				},
			},
			Spec: policyv1beta1.PodSecurityPolicySpec{
				HostPID:                true,
				ReadOnlyRootFilesystem: true,
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
					policyv1beta1.HostPath,
				},
				AllowedHostPaths: []policyv1beta1.AllowedHostPath{
					{
						PathPrefix: "/",
					},
				},
			},
		}
		clusterRolePSP := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:psp:kube-system:registry-configuration-cleaner",
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
				Name:      "gardener.cloud:psp:registry-configuration-cleaner",
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

	mountPropagationHostToContainer := corev1.MountPropagationHostToContainer

	daemonSet := &appsv1.DaemonSet{
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
					AutomountServiceAccountToken: pointer.Bool(false),
					ServiceAccountName:           serviceAccountName,
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
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:             "host-root-volume",
									MountPath:        "/host",
									ReadOnly:         false,
									MountPropagation: &mountPropagationHostToContainer,
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

	objects = append(objects, daemonSet)

	registry := managedresources.NewRegistry(kubernetes.ShootScheme, kubernetes.ShootCodec, kubernetes.ShootSerializer)

	return registry.AddAllAndSerialize(objects...)
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
