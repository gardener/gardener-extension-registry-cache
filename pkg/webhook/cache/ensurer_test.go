// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"encoding/base64"
	"testing"

	extensionscontextwebhook "github.com/gardener/gardener/extensions/pkg/webhook/context"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	registryinstall "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/install"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/pkg/webhook/cache"
)

func TestRegistryCacheWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registry Cache Webhook Suite")
}

var _ = Describe("Ensurer", func() {
	const namespace = "shoot--foo--bar"

	var (
		logger  = logr.Discard()
		ctx     = context.Background()
		caCerts = []string{"/etc/containerd/certs.d/ca-bundle.pem"}

		decoder    runtime.Decoder
		fakeClient client.Client
	)

	BeforeEach(func() {
		scheme := runtime.NewScheme()
		Expect(extensionsv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		registryinstall.Install(scheme)

		decoder = serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
		fakeClient = fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	})

	Describe("#EnsureCRIConfig", func() {
		var (
			cluster   *extensions.Cluster
			extension *extensionsv1alpha1.Extension
			criConfig extensionsv1alpha1.CRIConfig
		)

		BeforeEach(func() {
			cluster = &extensions.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
				Shoot:      &gardencorev1beta1.Shoot{},
			}

			extension = &extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-cache",
					Namespace: namespace,
				},
				Status: extensionsv1alpha1.ExtensionStatus{
					DefaultStatus: extensionsv1alpha1.DefaultStatus{
						ProviderStatus: &runtime.RawExtension{
							Object: &v1alpha3.RegistryStatus{
								TypeMeta: metav1.TypeMeta{
									APIVersion: v1alpha3.SchemeGroupVersion.String(),
									Kind:       "RegistryStatus",
								},
								Caches: []v1alpha3.RegistryCacheStatus{
									{
										Upstream:  "docker.io",
										Endpoint:  "https://10.0.0.1:5000",
										RemoteURL: "https://registry-1.docker.io",
									},
									{
										Upstream:  "europe-docker.pkg.dev",
										Endpoint:  "http://10.0.0.2:5000",
										RemoteURL: "https://europe-docker.pkg.dev",
									},
									{
										Upstream:  "my-registry.io:5000",
										Endpoint:  "https://10.0.0.3:5000",
										RemoteURL: "http://my-registry.io:5000",
									},
								},
							},
						},
					},
				},
			}

			criConfig = extensionsv1alpha1.CRIConfig{
				Containerd: &extensionsv1alpha1.ContainerdConfig{
					Registries: []extensionsv1alpha1.RegistryConfig{
						{
							Upstream: "foo.io",
							Server:   ptr.To("https://foo.io"),
							Hosts: []extensionsv1alpha1.RegistryHost{
								{
									URL:          "https://bar.io",
									Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability},
								},
							},
						},
					},
				},
			}
		})

		It("should return err when it fails to get the cluster", func() {
			osc := &extensionsv1alpha1.OperatingSystemConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
			}
			gctx := extensionscontextwebhook.NewGardenContext(fakeClient, osc)

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to get the cluster resource: could not get cluster for namespace '%s'", namespace)))
		})

		It("should do nothing if the shoot has a deletion timestamp set", func() {
			deletionTimestamp := metav1.Now()
			cluster.Shoot.DeletionTimestamp = &deletionTimestamp

			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)
			expectedContainerd := criConfig.Containerd.DeepCopy()

			Expect(ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)).To(Succeed())
			Expect(criConfig.Containerd).To(Equal(expectedContainerd))
		})

		It("should do nothing if hibernation is enabled for Shoot", func() {
			cluster.Shoot.Spec.Hibernation = &gardencorev1beta1.Hibernation{Enabled: ptr.To(true)}

			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)
			expectedContainerd := criConfig.Containerd.DeepCopy()

			Expect(ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)).To(Succeed())
			Expect(criConfig.Containerd).To(Equal(expectedContainerd))
		})

		It("return err when it fails to get the extension", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to get extension '%s/registry-cache'", namespace)))
		})

		It("should return err when extension .status.providerStatus is nil", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)
			extension.Status.ProviderStatus = nil

			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("extension '%s/registry-cache' does not have a .status.providerStatus specified", namespace)))
		})

		It("should return err when extension .status.providerStatus cannot be decoded", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)
			extension.Status.ProviderStatus = &runtime.RawExtension{Object: &corev1.Pod{}}

			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to decode providerStatus of extension '%s/registry-cache'", namespace)))
		})

		It("should add additional registry config to a nil containerd registry configs", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)
			criConfig.Containerd = nil

			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			expectedRegistries := []extensionsv1alpha1.RegistryConfig{
				createRegistryConfig("docker.io", "https://registry-1.docker.io", "https://10.0.0.1:5000", caCerts),
				createRegistryConfig("europe-docker.pkg.dev", "https://europe-docker.pkg.dev", "http://10.0.0.2:5000", nil),
				createRegistryConfig("my-registry.io:5000", "http://my-registry.io:5000", "https://10.0.0.3:5000", caCerts),
			}

			Expect(ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)).To(Succeed())
			Expect(criConfig.Containerd.Registries).To(ConsistOf(expectedRegistries))
		})

		It("should add additional registry config to the current containerd registry configs", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			expectedRegistries := criConfig.Containerd.DeepCopy().Registries
			expectedRegistries = append(expectedRegistries, []extensionsv1alpha1.RegistryConfig{
				createRegistryConfig("docker.io", "https://registry-1.docker.io", "https://10.0.0.1:5000", caCerts),
				createRegistryConfig("europe-docker.pkg.dev", "https://europe-docker.pkg.dev", "http://10.0.0.2:5000", nil),
				createRegistryConfig("my-registry.io:5000", "http://my-registry.io:5000", "https://10.0.0.3:5000", caCerts),
			}...)

			Expect(ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)).To(Succeed())
			Expect(criConfig.Containerd.Registries).To(ConsistOf(expectedRegistries))
		})

		It("should update existing registry config from containerd registry configs", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			expectedRegistries := criConfig.Containerd.DeepCopy().Registries
			expectedRegistries = append(expectedRegistries, []extensionsv1alpha1.RegistryConfig{
				createRegistryConfig("docker.io", "https://registry-1.docker.io", "https://10.0.0.1:5000", caCerts),
				createRegistryConfig("europe-docker.pkg.dev", "https://europe-docker.pkg.dev", "http://10.0.0.2:5000", nil),
				createRegistryConfig("my-registry.io:5000", "http://my-registry.io:5000", "https://10.0.0.3:5000", caCerts),
			}...)

			criConfig.Containerd.Registries = append(criConfig.Containerd.Registries, []extensionsv1alpha1.RegistryConfig{
				createRegistryConfig("docker.io", "foo", "bar", caCerts),
				createRegistryConfig("europe-docker.pkg.dev", "foo", "bar", nil),
				createRegistryConfig("my-registry.io:5000", "foo", "bar", caCerts),
			}...)

			Expect(ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)).To(Succeed())
			Expect(criConfig.Containerd.Registries).To(ConsistOf(expectedRegistries))
		})
	})

	Describe("#EnsureAdditionalFiles", func() {
		const caSecretName = "ca-extension-registry-cache-bundle-1a2b3c4d"

		var (
			cluster   *extensions.Cluster
			extension *extensionsv1alpha1.Extension
			caSecret  *corev1.Secret

			newFiles []extensionsv1alpha1.File
		)

		BeforeEach(func() {
			cluster = &extensions.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
				Shoot:      &gardencorev1beta1.Shoot{},
			}

			extension = &extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-cache",
					Namespace: cluster.ObjectMeta.Name,
				},
				Status: extensionsv1alpha1.ExtensionStatus{
					DefaultStatus: extensionsv1alpha1.DefaultStatus{
						ProviderStatus: &runtime.RawExtension{
							Object: &v1alpha3.RegistryStatus{
								TypeMeta: metav1.TypeMeta{
									APIVersion: v1alpha3.SchemeGroupVersion.String(),
									Kind:       "RegistryStatus",
								},
								CASecretName: ptr.To(caSecretName),
							},
						},
					},
				},
			}
			caSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      caSecretName,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"bundle.crt": []byte("bar"),
				},
			}

			newFiles = []extensionsv1alpha1.File{
				{
					Path: "/var/lib/foo/bar.txt",
					Content: extensionsv1alpha1.FileContent{
						Inline: &extensionsv1alpha1.FileContentInline{
							Data: "plain-text",
						},
					},
				},
			}
		})

		It("should return err when it fails to get the cluster", func() {
			osc := &extensionsv1alpha1.OperatingSystemConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
				},
			}
			gctx := extensionscontextwebhook.NewGardenContext(fakeClient, osc)

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalFiles(ctx, gctx, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to get the cluster resource: could not get cluster for namespace '%s'", namespace)))
		})

		It("should do nothing if the shoot has a deletion timestamp set", func() {
			deletionTimestamp := metav1.Now()
			cluster.Shoot.DeletionTimestamp = &deletionTimestamp

			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)
			expectedNewFiles := make([]extensionsv1alpha1.File, len(newFiles))
			copy(expectedNewFiles, newFiles)

			Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &newFiles, nil)).To(Succeed())
			Expect(newFiles).To(Equal(expectedNewFiles))
		})

		It("should do nothing if hibernation is enabled for Shoot", func() {
			cluster.Shoot.Spec.Hibernation = &gardencorev1beta1.Hibernation{Enabled: ptr.To(true)}

			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)
			expectedNewFiles := make([]extensionsv1alpha1.File, len(newFiles))
			copy(expectedNewFiles, newFiles)

			Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &newFiles, nil)).To(Succeed())
			Expect(newFiles).To(Equal(expectedNewFiles))
		})

		It("return err when it fails to get the extension", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalFiles(ctx, gctx, &newFiles, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to get extension '%s/registry-cache'", namespace)))
		})

		It("should return err when extension .status.providerStatus is nil", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)
			extension.Status.ProviderStatus = nil

			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalFiles(ctx, gctx, &newFiles, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("extension '%s/registry-cache' does not have a .status.providerStatus specified", namespace)))
		})

		It("should return err when extension .status.providerStatus cannot be decoded", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)
			extension.Status.ProviderStatus = &runtime.RawExtension{Object: &corev1.Pod{}}

			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalFiles(ctx, gctx, &newFiles, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to decode providerStatus of extension '%s/registry-cache'", namespace)))
		})

		It("should do nothing if .status.providerStatus.caSecretName is nil", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)
			extension.Status.ProviderStatus = &runtime.RawExtension{
				Object: &v1alpha3.RegistryStatus{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha3.SchemeGroupVersion.String(),
						Kind:       "RegistryStatus",
					},
					CASecretName: nil,
				},
			}

			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)
			expectedNewFiles := make([]extensionsv1alpha1.File, len(newFiles))
			copy(expectedNewFiles, newFiles)

			Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &newFiles, nil)).To(Succeed())
			Expect(newFiles).To(Equal(expectedNewFiles))
		})

		It("should return err when the CA bundle secret does not exist", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalFiles(ctx, gctx, &newFiles, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to get CA bundle secret '%s/%s'", namespace, caSecretName)))
		})

		It("should return err when the CA bundle secret does not contain the required field", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			caSecret.Data = map[string][]byte{
				"foo": []byte("bar"),
			}
			Expect(fakeClient.Create(ctx, caSecret)).To(Succeed())
			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalFiles(ctx, gctx, &newFiles, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to find 'bundle.crt' key in the CA bundle secret '%s/%s'", namespace, caSecretName)))
		})

		It("should add additional file to the current files", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			Expect(fakeClient.Create(ctx, caSecret)).To(Succeed())
			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)
			expectedNewFiles := make([]extensionsv1alpha1.File, len(newFiles))
			copy(expectedNewFiles, newFiles)
			expectedNewFiles = append(expectedNewFiles,
				extensionsv1alpha1.File{
					Path:        "/etc/containerd/certs.d/ca-bundle.pem",
					Permissions: ptr.To[uint32](0644),
					Content: extensionsv1alpha1.FileContent{
						Inline: &extensionsv1alpha1.FileContentInline{
							Encoding: "b64",
							Data:     base64.StdEncoding.EncodeToString([]byte("bar")),
						},
					},
				},
			)

			Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &newFiles, nil)).To(Succeed())
			Expect(newFiles).To(ConsistOf(expectedNewFiles))
		})

		It("should update file with the expected content if it already exists", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			Expect(fakeClient.Create(ctx, caSecret)).To(Succeed())
			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			newFiles = append(newFiles,
				extensionsv1alpha1.File{
					Path:        "/etc/containerd/certs.d/ca-bundle.pem",
					Permissions: ptr.To[uint32](0642),
					Content: extensionsv1alpha1.FileContent{
						Inline: &extensionsv1alpha1.FileContentInline{
							Encoding: "b64",
							Data:     base64.StdEncoding.EncodeToString([]byte("old-content")),
						},
					},
				},
			)
			expectedNewFiles := make([]extensionsv1alpha1.File, len(newFiles))
			copy(expectedNewFiles, newFiles)
			expectedNewFiles[1] = extensionsv1alpha1.File{
				Path:        "/etc/containerd/certs.d/ca-bundle.pem",
				Permissions: ptr.To[uint32](0644),
				Content: extensionsv1alpha1.FileContent{
					Inline: &extensionsv1alpha1.FileContentInline{
						Encoding: "b64",
						Data:     base64.StdEncoding.EncodeToString([]byte("bar")),
					},
				},
			}

			Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &newFiles, nil)).To(Succeed())
			Expect(newFiles).To(ConsistOf(expectedNewFiles))
		})
	})
})

func createRegistryConfig(upstream, server, host string, caCerts []string) extensionsv1alpha1.RegistryConfig {
	return extensionsv1alpha1.RegistryConfig{
		Upstream: upstream,
		Server:   ptr.To(server),
		Hosts: []extensionsv1alpha1.RegistryHost{
			{
				URL:          host,
				Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability, extensionsv1alpha1.ResolveCapability},
				CACerts:      caCerts,
			},
		},
		ReadinessProbe: ptr.To(true),
	}
}
