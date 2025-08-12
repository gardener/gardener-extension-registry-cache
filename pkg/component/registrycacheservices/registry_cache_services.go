// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registrycacheservices

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	registryapi "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/helper"
	"github.com/gardener/gardener-extension-registry-cache/pkg/constants"
	registryutils "github.com/gardener/gardener-extension-registry-cache/pkg/utils/registry"
)

const (
	managedResourceName = "extension-registry-cache-services"
)

// Values is a set of configuration values for the registry cache services.
type Values struct {
	// Caches are the registry caches to deploy.
	Caches []registryapi.RegistryCache
	// KeepObjectsOnDestroy marks whether the ManagedResource's .spec.keepObjects will be set to true
	// before ManagedResource deletion during the Destroy operation. When set to true, the deployed
	// resources by ManagedResources won't be deleted, but the ManagedResource itself will be deleted.
	KeepObjectsOnDestroy bool
}

// New creates a new instance of component.DeployWaiter for registry cache services.
func New(
	client client.Client,
	apiReader client.Reader,
	namespace string,
	values Values,
) component.DeployWaiter {
	return &registryCacheServices{
		client:    client,
		apiReader: apiReader,
		namespace: namespace,
		values:    values,
	}
}

type registryCacheServices struct {
	client    client.Client
	apiReader client.Reader
	namespace string
	values    Values
}

func (r *registryCacheServices) Deploy(ctx context.Context) error {
	data, err := r.computeResourcesData()
	if err != nil {
		return err
	}

	if err := managedresources.CreateForShoot(ctx, r.client, r.namespace, managedResourceName, "registry-cache", false, data); err != nil {
		return fmt.Errorf("failed to create ManagedResource for Shoot: %w", err)
	}

	return nil
}

func (r *registryCacheServices) Destroy(ctx context.Context) error {
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

func (r *registryCacheServices) Wait(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitForManagedResource)
	defer cancel()

	return managedresources.WaitUntilHealthy(timeoutCtx, r.apiReader, r.namespace, managedResourceName)
}

func (r *registryCacheServices) WaitCleanup(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitForManagedResource)
	defer cancel()

	return managedresources.WaitUntilDeleted(timeoutCtx, r.client, r.namespace, managedResourceName)
}

func (r *registryCacheServices) computeResourcesData() (map[string][]byte, error) {
	var services []client.Object

	for _, cache := range r.values.Caches {
		service := computeResourcesDataForService(&cache)

		services = append(services, service)
	}

	registry := managedresources.NewRegistry(kubernetes.ShootScheme, kubernetes.ShootCodec, kubernetes.ShootSerializer)

	return registry.AddAllAndSerialize(services...)
}

func computeResourcesDataForService(cache *registryapi.RegistryCache) *corev1.Service {
	var (
		upstreamLabel = registryutils.ComputeUpstreamLabelValue(cache.Upstream)
		name          = "registry-" + strings.ReplaceAll(upstreamLabel, ".", "-")
		remoteURL     = ptr.Deref(cache.RemoteURL, registryutils.GetUpstreamURL(cache.Upstream))
		scheme        = computeScheme(cache)
	)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceSystem,
			Labels:    registryutils.GetLabels(name, upstreamLabel),
			Annotations: map[string]string{
				constants.UpstreamAnnotation:  cache.Upstream,
				constants.RemoteURLAnnotation: remoteURL,
				constants.SchemeAnnotation:    scheme,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: registryutils.GetLabels(name, upstreamLabel),
			Ports: []corev1.ServicePort{
				{
					Name:       "registry-cache",
					Port:       constants.RegistryCacheServerPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("registry-cache"),
				},
				{
					Name:       "debug",
					Port:       constants.RegistryCacheDebugPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("debug"),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	return service
}

func computeScheme(cache *registryapi.RegistryCache) string {
	scheme := "http"
	if helper.TLSEnabled(cache) {
		scheme = "https"
	}

	return scheme
}
