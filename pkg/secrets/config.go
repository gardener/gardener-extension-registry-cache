// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package secrets

import (
	"net"
	"time"

	extensionssecretsmanager "github.com/gardener/gardener/extensions/pkg/util/secret/manager"
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
	secretutils "github.com/gardener/gardener/pkg/utils/secrets"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener-extension-registry-cache/pkg/constants"
	registryutils "github.com/gardener/gardener-extension-registry-cache/pkg/utils/registry"
)

const (
	// ManagerIdentity is the identity used for the Secrets Manager.
	ManagerIdentity = "extension-registry-cache"
	// CAName is the name of the CA secret.
	CAName = "ca-extension-registry-cache"
)

// ConfigsFor returns configurations for the secrets manager for the given registry caches services.
func ConfigsFor(services []corev1.Service) []extensionssecretsmanager.SecretConfigWithOptions {
	configs := []extensionssecretsmanager.SecretConfigWithOptions{
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:       CAName,
				CommonName: CAName,
				CertType:   secretutils.CACert,
				Validity:   ptr.To(730 * 24 * time.Hour),
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.Persist()},
		},
	}

	for _, service := range services {
		scheme := service.Annotations[constants.SchemeAnnotation]
		if scheme == "http" {
			continue
		}

		upstream := service.Annotations[constants.UpstreamAnnotation]
		name := TLSSecretNameForUpstream(upstream)

		configs = append(configs, extensionssecretsmanager.SecretConfigWithOptions{
			Config: &secretutils.CertificateSecretConfig{
				Name:                        name,
				CommonName:                  name,
				CertType:                    secretutils.ServerCert,
				DNSNames:                    kubernetesutils.DNSNamesForService(service.Name, metav1.NamespaceSystem),
				IPAddresses:                 []net.IP{net.ParseIP(service.Spec.ClusterIP)},
				Validity:                    ptr.To(90 * 24 * time.Hour),
				SkipPublishingCACertificate: true,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(CAName, secretsmanager.UseOldCA)},
		})
	}

	return configs
}

// TLSSecretNameForUpstream returns a TLS Secret name for a given upstream.
func TLSSecretNameForUpstream(upstream string) string {
	name := registryutils.ComputeKubernetesResourceName(upstream)
	return name + "-tls"
}
