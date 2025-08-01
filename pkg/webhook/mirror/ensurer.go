// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package mirror

import (
	"context"
	"fmt"
	"slices"

	extensionscontextwebhook "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mirrorapi "github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror"
	registryutils "github.com/gardener/gardener-extension-registry-cache/pkg/utils/registry"
)

// NewEnsurer creates a new mirror configuration ensurer.
func NewEnsurer(client client.Client, decoder runtime.Decoder, logger logr.Logger) genericmutator.Ensurer {
	return &ensurer{
		client:  client,
		decoder: decoder,
		logger:  logger.WithName("registry-mirror-ensurer"),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer

	client  client.Client
	decoder runtime.Decoder
	logger  logr.Logger
}

// EnsureCRIConfig ensures the CRI config.
func (e *ensurer) EnsureCRIConfig(ctx context.Context, gctx extensionscontextwebhook.GardenContext, newCRIConfig, _ *extensionsv1alpha1.CRIConfig) error {
	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the cluster resource: %w", err)
	}

	if cluster.Shoot.DeletionTimestamp != nil {
		e.logger.Info("Shoot has a deletion timestamp set, skipping the OperatingSystemConfig mutation", "shoot", client.ObjectKeyFromObject(cluster.Shoot))
		return nil
	}
	extension := &extensionsv1alpha1.Extension{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "registry-mirror",
			Namespace: cluster.ObjectMeta.Name,
		},
	}
	if err := e.client.Get(ctx, client.ObjectKeyFromObject(extension), extension); err != nil {
		return fmt.Errorf("failed to get extension '%s': %w", client.ObjectKeyFromObject(extension), err)
	}

	if extension.Spec.ProviderConfig == nil {
		return fmt.Errorf("extension '%s' does not have a .spec.providerConfig specified", client.ObjectKeyFromObject(extension))
	}

	mirrorConfig := &mirrorapi.MirrorConfig{}
	if err := runtime.DecodeInto(e.decoder, extension.Spec.ProviderConfig.Raw, mirrorConfig); err != nil {
		return fmt.Errorf("failed to decode providerConfig of extension '%s': %w", client.ObjectKeyFromObject(extension), err)
	}

	if newCRIConfig.Containerd == nil {
		newCRIConfig.Containerd = &extensionsv1alpha1.ContainerdConfig{}
	}

	for _, mirror := range mirrorConfig.Mirrors {
		cfg := extensionsv1alpha1.RegistryConfig{
			Upstream: mirror.Upstream,
			Server:   ptr.To(registryutils.GetUpstreamURL(mirror.Upstream)),
		}
		for _, host := range mirror.Hosts {
			registryHost := extensionsv1alpha1.RegistryHost{
				URL: host.Host,
			}
			for _, c := range host.Capabilities {
				switch c {
				case mirrorapi.MirrorHostCapabilityPull:
					registryHost.Capabilities = append(registryHost.Capabilities, extensionsv1alpha1.PullCapability)
				case mirrorapi.MirrorHostCapabilityResolve:
					registryHost.Capabilities = append(registryHost.Capabilities, extensionsv1alpha1.ResolveCapability)
				}
			}
			cfg.Hosts = append(cfg.Hosts, registryHost)
		}
		i := slices.IndexFunc(newCRIConfig.Containerd.Registries, func(registryConfig extensionsv1alpha1.RegistryConfig) bool {
			return registryConfig.Upstream == cfg.Upstream
		})
		if i == -1 {
			newCRIConfig.Containerd.Registries = append(newCRIConfig.Containerd.Registries, cfg)
		} else {
			newCRIConfig.Containerd.Registries[i] = cfg
		}
	}

	return nil
}
