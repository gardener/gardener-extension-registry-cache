// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"context"
	"fmt"

	extensionsconfigv1alpha1 "github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1"
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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-registry-cache/imagevector"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/config"
	registryapi "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/pkg/component/registrycaches"
	"github.com/gardener/gardener-extension-registry-cache/pkg/component/registrycacheservices"
	"github.com/gardener/gardener-extension-registry-cache/pkg/constants"
	"github.com/gardener/gardener-extension-registry-cache/pkg/secrets"
)

// NewActuator returns an actuator responsible for registry-cache Extension resources.
func NewActuator(client client.Client, apiReader client.Reader, decoder runtime.Decoder, config config.Configuration) extension.Actuator {
	return &actuator{
		client:    client,
		apiReader: apiReader,
		decoder:   decoder,
		config:    config,
	}
}

type actuator struct {
	client    client.Client
	apiReader client.Reader
	decoder   runtime.Decoder
	config    config.Configuration
}

// Reconcile the Extension resource.
func (a *actuator) Reconcile(ctx context.Context, logger logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()
	cluster, err := extensionscontroller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	if v1beta1helper.HibernationIsEnabled(cluster.Shoot) {
		return nil
	}

	if ex.Spec.ProviderConfig == nil {
		return fmt.Errorf("providerConfig is required for the registry-cache extension")
	}

	registryConfig := &registryapi.RegistryConfig{}
	if err := runtime.DecodeInto(a.decoder, ex.Spec.ProviderConfig.Raw, registryConfig); err != nil {
		return fmt.Errorf("failed to decode provider config: %w", err)
	}

	registryCacheServices := registrycacheservices.New(a.client, a.apiReader, namespace, registrycacheservices.Values{
		Caches: registryConfig.Caches,
	})

	if err = registryCacheServices.Deploy(ctx); err != nil {
		return fmt.Errorf("failed to deploy the registry cache services component: %w", err)
	}

	if err := registryCacheServices.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait the registry cache services component to be healthy: %w", err)
	}

	services, err := a.fetchRegistryCacheServices(ctx, namespace, registryConfig)
	if err != nil {
		return fmt.Errorf("failed to fetch registry cache Services: %w", err)
	}

	secretConfigs := secrets.ConfigsFor([]corev1.Service{})
	secretsManager, err := extensionssecretsmanager.SecretsManagerForCluster(ctx, logger.WithName("secretsmanager"), clock.RealClock{}, a.client, cluster, secrets.ManagerIdentity, secretConfigs)
	if err != nil {
		return err
	}

	image, err := imagevector.ImageVector().FindImage("registry")
	if err != nil {
		return fmt.Errorf("failed to find the registry image: %w", err)
	}

	registryCaches := registrycaches.New(a.client, namespace, secretsManager, registrycaches.Values{
		Image:              image.String(),
		VPAEnabled:         v1beta1helper.ShootWantsVerticalPodAutoscaler(cluster.Shoot),
		Services:           services,
		Caches:             registryConfig.Caches,
		ResourceReferences: cluster.Shoot.Spec.Resources,
	})

	if err = registryCaches.Deploy(ctx); err != nil {
		return fmt.Errorf("failed to deploy the registry caches component: %w", err)
	}

	registryStatus := computeProviderStatus(services, registryCaches.CASecretName())

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

	secretsManager, err := extensionssecretsmanager.SecretsManagerForCluster(ctx, logger.WithName("secretsmanager"), clock.RealClock{}, a.client, cluster, secrets.ManagerIdentity, nil)
	if err != nil {
		return err
	}

	registryCacheServices := registrycacheservices.New(a.client, a.apiReader, namespace, registrycacheservices.Values{})
	if err := component.OpDestroyAndWait(registryCacheServices).Destroy(ctx); err != nil {
		return fmt.Errorf("failed to destroy the registry cache services component: %w", err)
	}

	registryCaches := registrycaches.New(a.client, namespace, secretsManager, registrycaches.Values{})
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
func (a *actuator) Migrate(ctx context.Context, _ logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	registryCacheServices := registrycacheservices.New(a.client, a.apiReader, namespace, registrycacheservices.Values{
		KeepObjectsOnDestroy: true,
	})
	if err := component.OpDestroyAndWait(registryCacheServices).Destroy(ctx); err != nil {
		return fmt.Errorf("failed to destroy the registry cache services component: %w", err)
	}

	registryCaches := registrycaches.New(a.client, namespace, nil, registrycaches.Values{
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

	secretsManager, err := extensionssecretsmanager.SecretsManagerForCluster(ctx, logger.WithName("secretsmanager"), clock.RealClock{}, a.client, cluster, secrets.ManagerIdentity, nil)
	if err != nil {
		return err
	}

	registryCacheServices := registrycacheservices.New(a.client, a.apiReader, namespace, registrycacheservices.Values{})
	if err := registryCacheServices.Destroy(ctx); err != nil {
		return fmt.Errorf("failed to destroy the registry cache services component: %w", err)
	}

	registryCaches := registrycaches.New(a.client, namespace, secretsManager, registrycaches.Values{})
	if err := registryCaches.Destroy(ctx); err != nil {
		return fmt.Errorf("failed to destroy the registry caches component: %w", err)
	}

	return secretsManager.Cleanup(ctx)
}

func (a *actuator) fetchRegistryCacheServices(ctx context.Context, namespace string, registryConfig *registryapi.RegistryConfig) ([]corev1.Service, error) {
	_, shootClient, err := util.NewClientForShoot(ctx, a.client, namespace, client.Options{}, extensionsconfigv1alpha1.RESTOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create shoot client: %w", err)
	}

	selector := labels.NewSelector()
	requirement, err := labels.NewRequirement(constants.UpstreamHostLabel, selection.Exists, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create label selector: %w", err)
	}
	selector = selector.Add(*requirement)

	serviceList := &corev1.ServiceList{}
	if err := shootClient.List(ctx, serviceList, client.InNamespace(metav1.NamespaceSystem), client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return nil, fmt.Errorf("failed to read services from shoot: %w", err)
	}

	if len(serviceList.Items) != len(registryConfig.Caches) {
		return nil, fmt.Errorf("not all services for all configured caches exist")
	}

	return serviceList.Items, nil
}

func computeProviderStatus(services []corev1.Service, caSecretName *string) *v1alpha3.RegistryStatus {
	caches := make([]v1alpha3.RegistryCacheStatus, 0, len(services))
	for _, service := range services {
		caches = append(caches, v1alpha3.RegistryCacheStatus{
			Upstream:  service.Annotations[constants.UpstreamAnnotation],
			Endpoint:  fmt.Sprintf("%s://%s:%d", service.Annotations[constants.SchemeAnnotation], service.Spec.ClusterIP, constants.RegistryCachePort),
			RemoteURL: service.Annotations[constants.RemoteURLAnnotation],
		})
	}

	return &v1alpha3.RegistryStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha3.SchemeGroupVersion.String(),
			Kind:       "RegistryStatus",
		},
		Caches:       caches,
		CASecretName: caSecretName,
	}
}

func (a *actuator) updateProviderStatus(ctx context.Context, ex *extensionsv1alpha1.Extension, registryStatus *v1alpha3.RegistryStatus) error {
	patch := client.MergeFrom(ex.DeepCopy())
	ex.Status.ProviderStatus = &runtime.RawExtension{Object: registryStatus}
	return a.client.Status().Patch(ctx, ex, patch)
}
