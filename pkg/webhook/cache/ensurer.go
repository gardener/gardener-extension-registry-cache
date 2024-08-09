// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"fmt"
	"slices"

	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
)

// NewEnsurer creates a new registry cache ensurer.
func NewEnsurer(client client.Client, decoder runtime.Decoder, logger logr.Logger) genericmutator.Ensurer {
	return &ensurer{
		client:  client,
		decoder: decoder,
		logger:  logger.WithName("registry-cache-ensurer"),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	client  client.Client
	decoder runtime.Decoder
	logger  logr.Logger
}

// EnsureCRIConfig ensures the CRI config.
func (e *ensurer) EnsureCRIConfig(ctx context.Context, gctx gcontext.GardenContext, new, _ *extensionsv1alpha1.CRIConfig) error {
	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the cluster resource: %w", err)
	}

	if cluster.Shoot.DeletionTimestamp != nil {
		e.logger.Info("Shoot has a deletion timestamp set, skipping the OperatingSystemConfig mutation", "shoot", client.ObjectKeyFromObject(cluster.Shoot))
		return nil
	}
	// If hibernation is enabled for Shoot, then the .status.providerStatus field of the registry-cache Extension can be missing (on Shoot creation)
	// or outdated (if for hibernated Shoot a new registry is added). Hence, we skip the OperatingSystemConfig mutation when hibernation is enabled.
	// When Shoot is waking up, then .status.providerStatus will be updated in the Extension and the OperatingSystemConfig will be mutated according to it.
	if v1beta1helper.HibernationIsEnabled(cluster.Shoot) {
		e.logger.Info("Hibernation is enabled for Shoot, skipping the OperatingSystemConfig mutation", "shoot", client.ObjectKeyFromObject(cluster.Shoot))
		return nil
	}

	extension := &extensionsv1alpha1.Extension{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "registry-cache",
			Namespace: cluster.ObjectMeta.Name,
		},
	}
	if err := e.client.Get(ctx, client.ObjectKeyFromObject(extension), extension); err != nil {
		return fmt.Errorf("failed to get extension '%s': %w", client.ObjectKeyFromObject(extension), err)
	}

	if extension.Status.ProviderStatus == nil {
		return fmt.Errorf("extension '%s' does not have a .status.providerStatus specified", client.ObjectKeyFromObject(extension))
	}

	registryStatus := &api.RegistryStatus{}
	if _, _, err := e.decoder.Decode(extension.Status.ProviderStatus.Raw, nil, registryStatus); err != nil {
		return fmt.Errorf("failed to decode providerStatus of extension '%s': %w", client.ObjectKeyFromObject(extension), err)
	}

	if new.Containerd == nil {
		new.Containerd = &extensionsv1alpha1.ContainerdConfig{}
	}

	for _, cache := range registryStatus.Caches {
		cfg := extensionsv1alpha1.RegistryConfig{
			Upstream: cache.Upstream,
			Server:   ptr.To(cache.RemoteURL),
			Hosts: []extensionsv1alpha1.RegistryHost{{
				URL:          cache.Endpoint,
				Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability, extensionsv1alpha1.ResolveCapability},
			}},
			ReadinessProbe: ptr.To(true),
		}
		i := slices.IndexFunc(new.Containerd.Registries, func(registryConfig extensionsv1alpha1.RegistryConfig) bool {
			return registryConfig.Upstream == cfg.Upstream
		})
		if i == -1 {
			new.Containerd.Registries = append(new.Containerd.Registries, cfg)
		} else {
			new.Containerd.Registries[i] = cfg
		}
	}

	return nil
}
