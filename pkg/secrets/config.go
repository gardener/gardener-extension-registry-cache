package secrets

import (
	"net"
	"time"

	extensionssecretsmanager "github.com/gardener/gardener/extensions/pkg/util/secret/manager"
	secretutils "github.com/gardener/gardener/pkg/utils/secrets"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener-extension-registry-cache/pkg/constants"
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
		configs = append(configs, extensionssecretsmanager.SecretConfigWithOptions{
			Config: &secretutils.CertificateSecretConfig{
				Name:                        service.Annotations[constants.UpstreamAnnotation] + "-tls",
				CommonName:                  service.Annotations[constants.UpstreamAnnotation] + "-tls",
				CertType:                    secretutils.ServerCert,
				IPAddresses:                 []net.IP{net.ParseIP(service.Spec.ClusterIP).To4()},
				Validity:                    ptr.To(90 * 24 * time.Hour),
				SkipPublishingCACertificate: true,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(CAName, secretsmanager.UseOldCA)},
		})
	}
	return configs
}
