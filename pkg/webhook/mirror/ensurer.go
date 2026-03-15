// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package mirror

import (
	"context"
	"encoding/base64"
	"fmt"
	"path"
	"slices"
	"strings"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	extensionscontextwebhook "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	v1beta1helper "github.com/gardener/gardener/pkg/api/core/v1beta1/helper"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
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

	if !e.shouldMutate(cluster) {
		return nil
	}

	mirrorConfig, err := e.getProviderConfig(ctx, cluster)
	if err != nil {
		return err
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
			if host.CABundleSecretReferenceName != nil {
				registryHost.CACerts = []string{caBundlePath(mirror.Upstream, host.Host)}
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

// EnsureAdditionalFiles ensures that the mirror host's CA bundle is added to the <new> files.
func (e *ensurer) EnsureAdditionalFiles(ctx context.Context, gctx extensionscontextwebhook.GardenContext, newFiles, _ *[]extensionsv1alpha1.File) error {
	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the cluster resource: %w", err)
	}

	if !e.shouldMutate(cluster) {
		return nil
	}

	mirrorConfig, err := e.getProviderConfig(ctx, cluster)
	if err != nil {
		return err
	}

	for _, mirror := range mirrorConfig.Mirrors {
		for _, host := range mirror.Hosts {
			if host.CABundleSecretReferenceName != nil {
				ref := v1beta1helper.GetResourceByName(cluster.Shoot.Spec.Resources, *host.CABundleSecretReferenceName)
				if ref == nil || ref.ResourceRef.Kind != "Secret" {
					return fmt.Errorf("failed to find referenced resource with name %s and kind Secret", *host.CABundleSecretReferenceName)
				}

				refSecret := &corev1.Secret{}
				if err := extensionscontroller.GetObjectByReference(ctx, e.client, &ref.ResourceRef, cluster.ObjectMeta.Name, refSecret); err != nil {
					return fmt.Errorf("failed to read referenced secret %s%s for reference %s", v1beta1constants.ReferencedResourcesPrefix, ref.ResourceRef.Name, *host.CABundleSecretReferenceName)
				}

				caBundle, ok := refSecret.Data["bundle.crt"]
				if !ok {
					return fmt.Errorf("failed to find 'bundle.crt' key in the CA bundle secret '%s'", client.ObjectKeyFromObject(refSecret))
				}

				*newFiles = extensionswebhook.EnsureFileWithPath(*newFiles, extensionsv1alpha1.File{
					Path:        caBundlePath(mirror.Upstream, host.Host),
					Permissions: ptr.To[uint32](0644),
					Content: extensionsv1alpha1.FileContent{
						Inline: &extensionsv1alpha1.FileContentInline{
							Encoding: "b64",
							Data:     base64.StdEncoding.EncodeToString(caBundle),
						},
					},
				})
			}
		}
	}

	return nil
}

func (e *ensurer) shouldMutate(cluster *extensionscontroller.Cluster) bool {
	if cluster.Shoot.DeletionTimestamp != nil {
		e.logger.Info("Shoot has a deletion timestamp set, skipping the OperatingSystemConfig mutation", "shoot", client.ObjectKeyFromObject(cluster.Shoot))
		return false
	}

	return true
}

func (e *ensurer) getProviderConfig(ctx context.Context, cluster *extensionscontroller.Cluster) (*mirrorapi.MirrorConfig, error) {
	extension := &extensionsv1alpha1.Extension{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "registry-mirror",
			Namespace: cluster.ObjectMeta.Name,
		},
	}
	if err := e.client.Get(ctx, client.ObjectKeyFromObject(extension), extension); err != nil {
		return nil, fmt.Errorf("failed to get extension '%s': %w", client.ObjectKeyFromObject(extension), err)
	}

	if extension.Spec.ProviderConfig == nil {
		return nil, fmt.Errorf("extension '%s' does not have a .spec.providerConfig specified", client.ObjectKeyFromObject(extension))
	}

	mirrorConfig := &mirrorapi.MirrorConfig{}
	if err := runtime.DecodeInto(e.decoder, extension.Spec.ProviderConfig.Raw, mirrorConfig); err != nil {
		return nil, fmt.Errorf("failed to decode providerConfig of extension '%s': %w", client.ObjectKeyFromObject(extension), err)
	}
	return mirrorConfig, nil
}

func caBundlePath(upstream, host string) string {
	const baseDir = "/etc/containerd/certs.d"

	sanitizedUpstream := sanitizeUpstream(upstream)
	sanitizedHost := sanitizeHost(host)

	return path.Join(baseDir, sanitizedUpstream, sanitizedHost+"-ca-bundle.pem")
}

func sanitizeUpstream(upstream string) string {
	return strings.ReplaceAll(upstream, ":", "-")
}

func sanitizeHost(host string) string {
	sanitizedHost := strings.TrimPrefix(host, "https://")
	sanitizedHost = strings.TrimPrefix(sanitizedHost, "http://")
	sanitizedHost = strings.ReplaceAll(sanitizedHost, ":", "-")

	return sanitizedHost
}
