// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"context"
	"fmt"

	extensionsconfig "github.com/gardener/gardener/extensions/pkg/apis/config"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	"github.com/gardener/gardener/extensions/pkg/util"
	extensionssecretsmanager "github.com/gardener/gardener/extensions/pkg/util/secret/manager"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/component"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-registry-cache/imagevector"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/config"
	api "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/pkg/component/registrycaches"
	"github.com/gardener/gardener-extension-registry-cache/pkg/secrets"
)

// NewActuator returns an actuator responsible for registry-cache Extension resources.
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
func (a *actuator) Reconcile(ctx context.Context, logger logr.Logger, ex *extensionsv1alpha1.Extension) error {
	if ex.Spec.ProviderConfig == nil {
		return fmt.Errorf("providerConfig is required for the registry-cache extension")
	}

	registryConfig := &api.RegistryConfig{}
	if _, _, err := a.decoder.Decode(ex.Spec.ProviderConfig.Raw, nil, registryConfig); err != nil {
		return fmt.Errorf("failed to decode provider config: %w", err)
	}

	namespace := ex.GetNamespace()
	cluster, err := extensionscontroller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	if v1beta1helper.HibernationIsEnabled(cluster.Shoot) {
		return nil
	}

	_, shootClient, err := util.NewClientForShoot(ctx, a.client, namespace, client.Options{}, extensionsconfig.RESTOptions{})
	if err != nil {
		return fmt.Errorf("failed to create shoot client: %w", err)
	}

	secretsManager, err := extensionssecretsmanager.SecretsManagerForCluster(ctx, logger.WithName("secretsmanager"), clock.RealClock{}, a.client, cluster, secrets.ManagerIdentity, secrets.ConfigsFor([]corev1.Service{}))
	if err != nil {
		return err
	}

	var cacheStatuses []api.RegistryCacheStatus
	if ex.Status.ProviderStatus != nil {
		status := &api.RegistryStatus{}
		if _, _, err := a.decoder.Decode(ex.Status.ProviderStatus.Raw, nil, status); err != nil {
			return fmt.Errorf("failed to decode providerStatus of extension '%s': %w", client.ObjectKeyFromObject(ex), err)
		}
		cacheStatuses = status.Caches
	}

	image, err := imagevector.ImageVector().FindImage("registry")
	if err != nil {
		return fmt.Errorf("failed to find the registry image: %w", err)
	}

	// an empty status to fill
	registryStatus := &v1alpha3.RegistryStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha3.SchemeGroupVersion.String(),
			Kind:       "RegistryStatus",
		},
	}

	registryCaches := registrycaches.New(a.client, shootClient, secretsManager, logger, namespace, registrycaches.Values{
		Image:              image.String(),
		VPAEnabled:         v1beta1helper.ShootWantsVerticalPodAutoscaler(cluster.Shoot),
		Caches:             registryConfig.Caches,
		ResourceReferences: cluster.Shoot.Spec.Resources,
		CacheStatuses:      cacheStatuses,
		RegistryStatus:     registryStatus,
	})

	if err = registryCaches.Deploy(ctx); err != nil {
		return fmt.Errorf("failed to deploy the registry caches component: %w", err)
	}

	if err = a.updateProviderStatus(ctx, ex, registryStatus); err != nil {
		return fmt.Errorf("failed to update Extension status: %w", err)
	}

	if err = secretsManager.Cleanup(ctx); err != nil {
		return fmt.Errorf("failed to cleanup secrets: %w", err)
	}

	return nil
}

// Delete the Extension resource.
func (a *actuator) Delete(ctx context.Context, logger logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()
	cluster, err := extensionscontroller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	_, shootClient, err := util.NewClientForShoot(ctx, a.client, namespace, client.Options{}, extensionsconfig.RESTOptions{})
	if err != nil {
		return fmt.Errorf("failed to create shoot client: %w", err)
	}

	secretsManager, err := extensionssecretsmanager.SecretsManagerForCluster(ctx, logger.WithName("secretsmanager"), clock.RealClock{}, a.client, cluster, secrets.ManagerIdentity, nil)
	if err != nil {
		return err
	}

	registryCaches := registrycaches.New(a.client, shootClient, secretsManager, logger, namespace, registrycaches.Values{})
	if err := component.OpDestroyAndWait(registryCaches).Destroy(ctx); err != nil {
		return fmt.Errorf("failed to destroy the registry caches component: %w", err)
	}

	return secretsManager.Cleanup(ctx)
}

// Restore the Extension resource.
func (a *actuator) Restore(ctx context.Context, logger logr.Logger, ex *extensionsv1alpha1.Extension) error {
	return a.Reconcile(ctx, logger, ex)
}

// Migrate the Extension resource.
func (a *actuator) Migrate(ctx context.Context, logger logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	registryCaches := registrycaches.New(a.client, nil, nil, logger, namespace, registrycaches.Values{
		KeepObjectsOnDestroy: true,
	})
	if err := component.OpDestroyAndWait(registryCaches).Destroy(ctx); err != nil {
		return fmt.Errorf("failed to destroy the registry caches component: %w", err)
	}

	return nil
}

// ForceDelete the Extension resource.
//
// We don't need to wait for the ManagedResource deletion because ManagedResources are finalized by gardenlet
// in later step in the Shoot force deletion flow.
func (a *actuator) ForceDelete(ctx context.Context, logger logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()
	cluster, err := extensionscontroller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	_, shootClient, err := util.NewClientForShoot(ctx, a.client, namespace, client.Options{}, extensionsconfig.RESTOptions{})
	if err != nil {
		return fmt.Errorf("failed to create shoot client: %w", err)
	}

	secretsManager, err := extensionssecretsmanager.SecretsManagerForCluster(ctx, logger.WithName("secretsmanager"), clock.RealClock{}, a.client, cluster, secrets.ManagerIdentity, nil)
	if err != nil {
		return err
	}

	registryCaches := registrycaches.New(a.client, shootClient, secretsManager, logger, namespace, registrycaches.Values{})
	if err := component.OpDestroy(registryCaches).Destroy(ctx); err != nil {
		return fmt.Errorf("failed to destroy the registry caches component: %w", err)
	}

	return secretsManager.Cleanup(ctx)
}

func (a *actuator) updateProviderStatus(ctx context.Context, ex *extensionsv1alpha1.Extension, registryStatus *v1alpha3.RegistryStatus) error {
	patch := client.MergeFrom(ex.DeepCopy())
	ex.Status.ProviderStatus = &runtime.RawExtension{Object: registryStatus}
	return a.client.Status().Patch(ctx, ex, patch)
}
