// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package mirror

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	extensionscontextwebhook "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pelletier/go-toml/v2"
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

	mirrorConfig, err := e.getMirrorConfigForCluster(ctx, cluster)
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
			if len(host.CABundle) > 0 {
				registryHost.CACerts = append(registryHost.CACerts, caFilename(mirror.Upstream, host.Host))
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
func (e *ensurer) EnsureAdditionalFiles(ctx context.Context, gctx extensionscontextwebhook.GardenContext, newObj, _ *[]extensionsv1alpha1.File) error {
	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the cluster resource: %w", err)
	}

	if cluster.Shoot.DeletionTimestamp != nil {
		e.logger.Info("Shoot has a deletion timestamp set, skipping the OperatingSystemConfig mutation", "shoot", client.ObjectKeyFromObject(cluster.Shoot))
		return nil
	}

	mirrorConfig, err := e.getMirrorConfigForCluster(ctx, cluster)
	if err != nil {
		return err
	}

	for _, mirror := range mirrorConfig.Mirrors {
		for _, caFile := range getCAFiles(mirror) {
			*newObj = extensionswebhook.EnsureFileWithPath(*newObj, caFile)
		}
	}

	return nil
}

func (e *ensurer) EnsureAdditionalProvisionFiles(ctx context.Context, gctx extensionscontextwebhook.GardenContext, newObj, _ *[]extensionsv1alpha1.File) error {
	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the cluster resource: %w", err)
	}

	if cluster.Shoot.DeletionTimestamp != nil {
		e.logger.Info("Shoot has a deletion timestamp set, skipping the OperatingSystemConfig mutation", "shoot", client.ObjectKeyFromObject(cluster.Shoot))
		return nil
	}

	mirrorConfig, err := e.getMirrorConfigForCluster(ctx, cluster)
	if err != nil {
		return err
	}

	for _, mirror := range mirrorConfig.Mirrors {
		if !mirror.ProvisionRelevant {
			continue
		}
		err = ensureHostsConfig(mirror, newObj)
		if err != nil {
			return err
		}
		for _, caFile := range getCAFiles(mirror) {
			*newObj = extensionswebhook.EnsureFileWithPath(*newObj, caFile)
		}
	}
	return nil
}

type containerdConfig struct {
	Server string                    `toml:"server" comment:"Created by gardener-extension-registry-mirror"`
	Host   map[string]containerdHost `toml:"host"`
}

type containerdHost struct {
	Capabilities []extensionsv1alpha1.RegistryCapability `toml:"capabilities"`
	CA           string                                  `toml:"ca,omitempty"`
}

func (e *ensurer) getMirrorConfigForCluster(ctx context.Context, cluster *controller.Cluster) (*mirrorapi.MirrorConfig, error) {
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

func ensureHostsConfig(config mirrorapi.MirrorConfiguration, files *[]extensionsv1alpha1.File) error {
	cConfig := containerdConfig{
		Server: registryutils.GetUpstreamURL(config.Upstream),
		Host:   map[string]containerdHost{},
	}

	for _, host := range config.Hosts {
		cHost := containerdHost{}
		for _, c := range host.Capabilities {
			switch c {
			case mirrorapi.MirrorHostCapabilityPull:
				cHost.Capabilities = append(cHost.Capabilities, extensionsv1alpha1.PullCapability)
			case mirrorapi.MirrorHostCapabilityResolve:
				cHost.Capabilities = append(cHost.Capabilities, extensionsv1alpha1.PullCapability)
			}
		}
		if len(host.CABundle) > 0 {
			cHost.CA = caFilename(config.Upstream, host.Host)
		}
		cConfig.Host[host.Host] = cHost
	}

	data, err := toml.Marshal(cConfig)
	if err != nil {
		return fmt.Errorf("failed to generate cConfig hosts.toml: %w", err)
	}

	hostsFile := extensionsv1alpha1.File{
		Path:        path.Join(configBaseDir(config.Upstream), "hosts.toml"),
		Permissions: ptr.To[uint32](0o644),
		Content: extensionsv1alpha1.FileContent{
			Inline: &extensionsv1alpha1.FileContentInline{
				Encoding: "",
				Data:     string(data),
			},
		},
	}
	*files = extensionswebhook.EnsureFileWithPath(*files, hostsFile)
	return nil
}

func getCAFiles(config mirrorapi.MirrorConfiguration) []extensionsv1alpha1.File {
	var files []extensionsv1alpha1.File
	for _, host := range config.Hosts {
		if len(host.CABundle) == 0 {
			continue
		}
		caFile := extensionsv1alpha1.File{
			Path:        caFilename(config.Upstream, host.Host),
			Permissions: ptr.To[uint32](0o644),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     string(host.CABundle),
				},
			},
		}
		files = extensionswebhook.EnsureFileWithPath(files, caFile)
	}
	return files
}

func caFilename(upstream, host string) string {
	return path.Join(configBaseDir(upstream), hostname(host)+".crt")
}

func hostname(h string) string {
	h = strings.TrimPrefix(h, "https://")
	h = strings.TrimPrefix(h, "http://")
	h = strings.TrimSuffix(h, "/")
	return h
}

func configBaseDir(server string) string {
	const baseDir = "/etc/containerd/certs.d"
	return path.Join(baseDir, hostname(server))
}
