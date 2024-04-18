// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"strings"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
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

var (
	//go:embed scripts/configure-containerd-registries.sh
	configureContainerdRegistriesScript string
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

// EnsureAdditionalFiles ensures that the configure-containerd-registries.sh script is added to the <new> files.
func (e *ensurer) EnsureAdditionalFiles(_ context.Context, _ gcontext.GardenContext, new, _ *[]extensionsv1alpha1.File) error {
	*new = extensionswebhook.EnsureFileWithPath(*new, extensionsv1alpha1.File{
		Path:        "/opt/bin/configure-containerd-registries.sh",
		Permissions: ptr.To(int32(0744)),
		Content: extensionsv1alpha1.FileContent{
			Inline: &extensionsv1alpha1.FileContentInline{
				Encoding: "b64",
				Data:     base64.StdEncoding.EncodeToString([]byte(configureContainerdRegistriesScript)),
			},
		},
	})

	return nil
}

// EnsureAdditionalUnits ensures that the configure-containerd-registries.service unit is added to the <new> units.
func (e *ensurer) EnsureAdditionalUnits(ctx context.Context, gctx gcontext.GardenContext, new, _ *[]extensionsv1alpha1.Unit) error {
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
		e.logger.Info("Hibernation is enabeld for Shoot, skipping the OperatingSystemConfig mutation", "shoot", client.ObjectKeyFromObject(cluster.Shoot))
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

	scriptArgs := make([]string, 0, len(registryStatus.Caches))
	for _, cache := range registryStatus.Caches {
		scriptArgs = append(scriptArgs, fmt.Sprintf("%s,%s,%s", cache.Upstream, cache.Endpoint, cache.RemoteURL))
	}

	unit := extensionsv1alpha1.Unit{
		Name:    "configure-containerd-registries.service",
		Command: ptr.To(extensionsv1alpha1.CommandStart),
		Enable:  ptr.To(true),
		Content: ptr.To(`[Unit]
Description=Configures containerd registries

[Install]
WantedBy=multi-user.target

[Unit]
After=containerd.service
Requires=containerd.service

[Service]
Type=simple
ExecStart=/opt/bin/configure-containerd-registries.sh ` + strings.Join(scriptArgs, " ")),
	}

	*new = extensionswebhook.EnsureUnitWithName(*new, unit)

	return nil
}
