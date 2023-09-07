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
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
	"github.com/gardener/gardener-extension-registry-cache/pkg/constants"
	registryutils "github.com/gardener/gardener-extension-registry-cache/pkg/utils/registry"
)

const (
	// ManagedResourceName is the ManagedResource name for the registry cache resources in the shoot.
	ManagedResourceName = "extension-registry-cache"
)

// Values is a set of configuration values for the registry caches.
type Values struct {
	// Image is the container image used for the registry cache.
	Image string
	// Caches are the registry caches to deploy.
	Caches []v1alpha1.RegistryCache
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
	data, err := r.computeResourcesData()
	if err != nil {
		return err
	}

	var (
		origin      = "registry-cache"
		keepObjects = false

		secretName, secret = managedresources.NewSecret(r.client, r.namespace, ManagedResourceName, data, false)
		managedResource    = managedresources.NewForShoot(r.client, r.namespace, ManagedResourceName, origin, keepObjects).
					WithSecretRef(secretName).
					DeletePersistentVolumeClaims(true)
	)

	if err := secret.Reconcile(ctx); err != nil {
		return fmt.Errorf("failed to create or update secret of managed resources: %w", err)
	}

	if err := managedResource.Reconcile(ctx); err != nil {
		return fmt.Errorf("failed to not create or update managed resource: %w", err)
	}

	return nil
}

// Destroy implements component.DeployWaiter.
func (r *registryCaches) Destroy(ctx context.Context) error {
	return managedresources.Delete(ctx, r.client, r.namespace, ManagedResourceName, false)
}

// TimeoutWaitForManagedResource is the timeout used while waiting for the ManagedResources to become healthy
// or deleted.
var TimeoutWaitForManagedResource = 2 * time.Minute

// Wait implements component.DeployWaiter.
func (r *registryCaches) Wait(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitForManagedResource)
	defer cancel()

	return managedresources.WaitUntilHealthy(timeoutCtx, r.client, r.namespace, ManagedResourceName)
}

// WaitCleanup implements component.DeployWaiter.
func (r *registryCaches) WaitCleanup(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitForManagedResource)
	defer cancel()

	return managedresources.WaitUntilDeleted(timeoutCtx, r.client, r.namespace, ManagedResourceName)
}

func (r *registryCaches) computeResourcesData() (map[string][]byte, error) {
	objects := []client.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: constants.NamespaceRegistryCache,
			},
		},
	}

	for _, cache := range r.values.Caches {
		cacheObjects, err := computeResourcesDataForRegistryCache(&cache, r.values.Image)
		if err != nil {
			return nil, fmt.Errorf("failed to compute resources for upstream %s: %w", cache.Upstream, err)
		}

		objects = append(objects, cacheObjects...)
	}

	registry := managedresources.NewRegistry(kubernetes.ShootScheme, kubernetes.ShootCodec, kubernetes.ShootSerializer)

	return registry.AddAllAndSerialize(objects...)
}

func computeResourcesDataForRegistryCache(cache *v1alpha1.RegistryCache, image string) ([]client.Object, error) {
	if cache.Size == nil {
		return nil, fmt.Errorf("registry cache size is required")
	}
	if cache.GarbageCollectionEnabled == nil {
		return nil, fmt.Errorf("registry cache garbageCollectionEnabled is required")
	}

	const registryCacheVolumeName = "cache-volume"

	var (
		name   = strings.Replace(fmt.Sprintf("registry-%s", strings.Split(cache.Upstream, ":")[0]), ".", "-", -1)
		labels = map[string]string{
			"app":                       name,
			constants.UpstreamHostLabel: cache.Upstream,
		}

		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: constants.NamespaceRegistryCache,
				Labels:    labels,
			},
			Spec: corev1.ServiceSpec{
				Selector: labels,
				Ports: []corev1.ServicePort{{
					Name:       "registry-cache",
					Port:       constants.RegistryCachePort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("registry-cache"),
				}},
				Type: corev1.ServiceTypeClusterIP,
			},
		}

		statefulSet = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: constants.NamespaceRegistryCache,
				Labels:    labels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: service.Name,
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Replicas: pointer.Int32(1),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "registry-cache",
								Image:           image,
								ImagePullPolicy: corev1.PullIfNotPresent,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: constants.RegistryCachePort,
										Name:          "registry-cache",
									},
								},
								Env: []corev1.EnvVar{
									{
										Name:  "REGISTRY_PROXY_REMOTEURL",
										Value: registryutils.GetUpstreamURL(cache.Upstream),
									},
									{
										Name:  "REGISTRY_STORAGE_DELETE_ENABLED",
										Value: strconv.FormatBool(*cache.GarbageCollectionEnabled),
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      registryCacheVolumeName,
										ReadOnly:  false,
										MountPath: "/var/lib/registry",
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
							Labels: labels,
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: *cache.Size,
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
