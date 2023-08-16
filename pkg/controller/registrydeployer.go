// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package controller

import (
	"fmt"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	registryutils "github.com/gardener/gardener-extension-registry-cache/pkg/utils/registry"
)

type registryCache struct {
	Name      string
	Namespace string
	Labels    map[string]string

	Upstream                 string
	VolumeSize               resource.Quantity
	GarbageCollectionEnabled bool

	RegistryImage string
}

const (
	registryCacheNamespaceName = "registry-cache"
	registryCacheInternalName  = "registry-cache"
	registryCacheVolumeName    = "cache-volume"
	registryVolumeMountPath    = "/var/lib/registry"
	// registryCachePort is the port on which the pull through cache serves requests.
	registryCachePort = 5000

	environmentVarialbleNameRegistryURL    = "REGISTRY_PROXY_REMOTEURL"
	environmentVarialbleNameRegistryDelete = "REGISTRY_STORAGE_DELETE_ENABLED"

	registryCacheServiceUpstreamLabel = "upstream-host"
)

func (c *registryCache) Ensure() ([]client.Object, error) {
	c.Name = strings.Replace(fmt.Sprintf("registry-%s", strings.Split(c.Upstream, ":")[0]), ".", "-", -1)

	if c.Labels == nil {
		c.Labels = map[string]string{
			"app": c.Name,
		}
	}

	c.Labels[registryCacheServiceUpstreamLabel] = c.Upstream

	var (
		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.Name,
				Namespace: registryCacheNamespaceName,
				Labels:    c.Labels,
			},
			Spec: corev1.ServiceSpec{
				Selector: c.Labels,
				Ports: []corev1.ServicePort{{
					Name:       registryCacheInternalName,
					Port:       registryCachePort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(registryCacheInternalName),
				}},
				Type: corev1.ServiceTypeClusterIP,
			},
		}

		statefulSet = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.Name,
				Namespace: registryCacheNamespaceName,
				Labels:    c.Labels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: service.Name,
				Selector: &metav1.LabelSelector{
					MatchLabels: c.Labels,
				},
				Replicas: pointer.Int32(1),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: c.Labels,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            registryCacheInternalName,
								Image:           c.RegistryImage,
								ImagePullPolicy: corev1.PullIfNotPresent,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: registryCachePort,
										Name:          registryCacheInternalName,
									},
								},
								Env: []corev1.EnvVar{
									{
										Name:  environmentVarialbleNameRegistryURL,
										Value: registryutils.GetUpstreamURL(c.Upstream),
									},
									{
										Name:  environmentVarialbleNameRegistryDelete,
										Value: strconv.FormatBool(c.GarbageCollectionEnabled),
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      registryCacheVolumeName,
										ReadOnly:  false,
										MountPath: registryVolumeMountPath,
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
							Labels: c.Labels,
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: c.VolumeSize,
								},
							},
						},
					},
				},
			},
		}
	)

	return []client.Object{
		service,
		statefulSet,
	}, nil
}
