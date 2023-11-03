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

package extension

import (
	"context"
	"fmt"

	extensionsconfig "github.com/gardener/gardener/extensions/pkg/apis/config"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	"github.com/gardener/gardener/extensions/pkg/util"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/component"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-registry-cache/imagevector"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/config"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
	"github.com/gardener/gardener-extension-registry-cache/pkg/component/registrycaches"
	"github.com/gardener/gardener-extension-registry-cache/pkg/component/registryconfigurationcleaner"
	"github.com/gardener/gardener-extension-registry-cache/pkg/constants"
)

// NewActuator returns an actuator responsible for Extension resources.
func NewActuator(client client.Client, decoder runtime.Decoder, config config.Configuration) extension.Actuator {
	return &actuator{
		client:  client,
		decoder: decoder,
		config:  config,
	}
}

type actuator struct {
	client  client.Client
	decoder runtime.Decoder
	config  config.Configuration
}

// Reconcile the Extension resource.
func (a *actuator) Reconcile(ctx context.Context, _ logr.Logger, ex *extensionsv1alpha1.Extension) error {
	if ex.Spec.ProviderConfig == nil {
		return fmt.Errorf("providerConfig is required for the registry-cache extension")
	}

	registryConfig := &v1alpha1.RegistryConfig{}
	if _, _, err := a.decoder.Decode(ex.Spec.ProviderConfig.Raw, nil, registryConfig); err != nil {
		return fmt.Errorf("failed to decode provider config: %w", err)
	}

	namespace := ex.GetNamespace()
	cluster, err := extensionscontroller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Clean registry configuration if a registry cache is removed.
	if ex.Status.ProviderStatus != nil {
		registryStatus := &v1alpha1.RegistryStatus{}
		if _, _, err := a.decoder.Decode(ex.Status.ProviderStatus.Raw, nil, registryStatus); err != nil {
			return fmt.Errorf("failed to decode providerStatus of extension '%s': %w", client.ObjectKeyFromObject(ex), err)
		}

		existingUpstreams := sets.New[string]()
		for _, cache := range registryStatus.Caches {
			existingUpstreams.Insert(cache.Upstream)
		}

		desiredUpstreams := sets.New[string]()
		for _, cache := range registryConfig.Caches {
			desiredUpstreams.Insert(cache.Upstream)
		}

		upstreamsToDelete := existingUpstreams.Difference(desiredUpstreams)
		if upstreamsToDelete.Len() > 0 {
			if err := cleanRegistryConfiguration(ctx, cluster, a.client, ex.GetNamespace(), false, upstreamsToDelete.UnsortedList()); err != nil {
				return err
			}
		}
	}

	image, err := imagevector.ImageVector().FindImage("registry")
	if err != nil {
		return fmt.Errorf("failed to find the registry image: %w", err)
	}

	registryCaches := registrycaches.New(a.client, namespace, registrycaches.Values{
		Image:      image.String(),
		VPAEnabled: v1beta1helper.ShootWantsVerticalPodAutoscaler(cluster.Shoot),
		Caches:     registryConfig.Caches,
	})

	if err := registryCaches.Deploy(ctx); err != nil {
		return fmt.Errorf("failed to deploy the registry caches component: %w", err)
	}

	// If the hibernation is enabled, don't try to fetch the registry cache endpoints from the Shoot cluster.
	if !v1beta1helper.HibernationIsEnabled(cluster.Shoot) {
		registryStatus, err := a.computeProviderStatus(ctx, registryConfig, namespace)
		if err != nil {
			return fmt.Errorf("failed to compute provider status: %w", err)
		}

		if err := a.updateProviderStatus(ctx, ex, registryStatus); err != nil {
			return fmt.Errorf("failed to update Extension status: %w", err)
		}
	}

	return nil
}

// Delete the Extension resource.
func (a *actuator) Delete(ctx context.Context, _ logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	if ex.Status.ProviderStatus != nil {
		registryStatus := &v1alpha1.RegistryStatus{}
		if _, _, err := a.decoder.Decode(ex.Status.ProviderStatus.Raw, nil, registryStatus); err != nil {
			return fmt.Errorf("failed to decode providerStatus of extension '%s': %w", client.ObjectKeyFromObject(ex), err)
		}

		cluster, err := extensionscontroller.GetCluster(ctx, a.client, namespace)
		if err != nil {
			return fmt.Errorf("failed to get cluster: %w", err)
		}

		upstreams := make([]string, 0, len(registryStatus.Caches))
		for _, cache := range registryStatus.Caches {
			upstreams = append(upstreams, cache.Upstream)
		}

		if err := cleanRegistryConfiguration(ctx, cluster, a.client, ex.GetNamespace(), true, upstreams); err != nil {
			return err
		}
	}

	registryCaches := registrycaches.New(a.client, namespace, registrycaches.Values{})
	if err := component.OpDestroyAndWait(registryCaches).Destroy(ctx); err != nil {
		return fmt.Errorf("failed to destroy the registry caches component: %w", err)
	}

	return nil
}

// Restore the Extension resource.
func (a *actuator) Restore(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	return a.Reconcile(ctx, log, ex)
}

// Migrate the Extension resource.
func (a *actuator) Migrate(_ context.Context, _ logr.Logger, _ *extensionsv1alpha1.Extension) error {
	return nil
}

// ForceDelete the Extension resource.
//
// We don't need to wait for the ManagedResource deletion because ManagedResources are finalized by gardenlet
// in later step in the Shoot force deletion flow.
func (a *actuator) ForceDelete(ctx context.Context, _ logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	registryCaches := registrycaches.New(a.client, namespace, registrycaches.Values{})
	if err := component.OpDestroy(registryCaches).Destroy(ctx); err != nil {
		return fmt.Errorf("failed to destroy the registry caches component: %w", err)
	}

	return nil
}

func (a *actuator) computeProviderStatus(ctx context.Context, registryConfig *v1alpha1.RegistryConfig, namespace string) (*v1alpha1.RegistryStatus, error) {
	// get service IPs from shoot
	_, shootClient, err := util.NewClientForShoot(ctx, a.client, namespace, client.Options{}, extensionsconfig.RESTOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create shoot client: %w", err)
	}

	selector := labels.NewSelector()
	r, err := labels.NewRequirement(constants.UpstreamHostLabel, selection.Exists, nil)
	if err != nil {
		return nil, err
	}
	selector = selector.Add(*r)

	// get all registry cache services
	services := &corev1.ServiceList{}
	if err := shootClient.List(ctx, services, client.InNamespace(metav1.NamespaceSystem), client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return nil, fmt.Errorf("failed to read services from shoot: %w", err)
	}

	if len(services.Items) != len(registryConfig.Caches) {
		return nil, fmt.Errorf("not all services for all configured caches exist")
	}

	caches := []v1alpha1.RegistryCacheStatus{}
	for _, service := range services.Items {
		caches = append(caches, v1alpha1.RegistryCacheStatus{
			Upstream: service.Labels[constants.UpstreamHostLabel],
			Endpoint: fmt.Sprintf("http://%s:%d", service.Spec.ClusterIP, constants.RegistryCachePort),
		})
	}

	return &v1alpha1.RegistryStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "RegistryStatus",
		},
		Caches: caches,
	}, nil
}

func (a *actuator) updateProviderStatus(ctx context.Context, ex *extensionsv1alpha1.Extension, registryStatus *v1alpha1.RegistryStatus) error {
	patch := client.MergeFrom(ex.DeepCopy())
	ex.Status.ProviderStatus = &runtime.RawExtension{Object: registryStatus}
	return a.client.Status().Patch(ctx, ex, patch)
}

func cleanRegistryConfiguration(ctx context.Context, cluster *extensionscontroller.Cluster, client client.Client, namespace string, deleteSystemdUnit bool, upstreams []string) error {
	// If the Shoot is hibernated, we don't have Nodes. Hence, there is no need to clean up anything.
	if extensionscontroller.IsHibernated(cluster) {
		return nil
	}

	alpineImage, err := imagevector.ImageVector().FindImage("alpine")
	if err != nil {
		return fmt.Errorf("failed to find the alpine image: %w", err)
	}
	pauseImage, err := imagevector.ImageVector().FindImage("pause")
	if err != nil {
		return fmt.Errorf("failed to find the pause image: %w", err)
	}

	values := registryconfigurationcleaner.Values{
		AlpineImage:       alpineImage.String(),
		PauseImage:        pauseImage.String(),
		DeleteSystemdUnit: deleteSystemdUnit,
		Upstreams:         upstreams,
	}
	cleaner := registryconfigurationcleaner.New(client, namespace, values)

	if err := component.OpWait(cleaner).Deploy(ctx); err != nil {
		return fmt.Errorf("failed to deploy the registry configuration cleaner component: %w", err)
	}

	if err := component.OpDestroyAndWait(cleaner).Destroy(ctx); err != nil {
		return fmt.Errorf("failed to destroy the registry configuration cleaner component: %w", err)
	}

	return nil
}
