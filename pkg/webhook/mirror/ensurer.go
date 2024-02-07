// Copyright (c) 2024 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package mirror

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror"
	registryutils "github.com/gardener/gardener-extension-registry-cache/pkg/utils/registry"
)

const (
	// containerdRegistryHostsDirectory is a directory that is created by the containerd-inializer systemd service.
	// containerd is configured to read registry configuration from this directory.
	containerdRegistryHostsDirectory = "/etc/containerd/certs.d"
)

var (
	//go:embed templates/hosts.toml.tpl
	hostsTOMLContentTpl string
	hostsTOMLTpl        *template.Template
)

func init() {
	var err error
	hostsTOMLTpl, err = template.
		New("hosts.toml.tpl").
		Funcs(sprig.TxtFuncMap()).
		Parse(hostsTOMLContentTpl)
	utilruntime.Must(err)
}

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

// EnsureAdditionalFiles ensures that the containerd registry configuration files are added to the <new> files.
func (e *ensurer) EnsureAdditionalFiles(ctx context.Context, gctx gcontext.GardenContext, new, _ *[]extensionsv1alpha1.File) error {
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

	mirrorConfig := &mirror.MirrorConfig{}
	if _, _, err := e.decoder.Decode(extension.Spec.ProviderConfig.Raw, nil, mirrorConfig); err != nil {
		return fmt.Errorf("failed to decode providerConfig of extension '%s': %w", client.ObjectKeyFromObject(extension), err)
	}

	for _, mirror := range mirrorConfig.Mirrors {
		var hostsTOML bytes.Buffer
		if err := hostsTOMLTpl.Execute(&hostsTOML, map[string]interface{}{
			"Server": registryutils.GetUpstreamURL(mirror.Upstream),
			"Hosts":  mirror.Hosts,
		}); err != nil {
			return fmt.Errorf("cannot execute hosts.toml template: %w", err)
		}

		*new = extensionswebhook.EnsureFileWithPath(*new, extensionsv1alpha1.File{
			Path:        filepath.Join(containerdRegistryHostsDirectory, mirror.Upstream, "hosts.toml"),
			Permissions: pointer.Int32(0644),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Data: hostsTOML.String(),
				},
			},
		})
	}

	return nil
}
