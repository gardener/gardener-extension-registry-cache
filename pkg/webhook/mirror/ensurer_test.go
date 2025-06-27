// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package mirror_test

import (
	"context"
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

	mirrorinstall "github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror/install"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror/v1alpha1"
	"github.com/gardener/gardener-extension-registry-cache/pkg/webhook/mirror"
)

func TestRegistryMirrorWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registry Mirror Webhook Suite")
}

var _ = Describe("Ensurer", func() {
	var (
		logger = logr.Discard()
		ctx    = context.Background()

		decoder    runtime.Decoder
		fakeClient client.Client
	)

	BeforeEach(func() {
		scheme := runtime.NewScheme()
		Expect(extensionsv1alpha1.AddToScheme(scheme)).To(Succeed())
		mirrorinstall.Install(scheme)

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
				ObjectMeta: metav1.ObjectMeta{Name: "shoot--foo--bar"},
				Shoot:      &gardencorev1beta1.Shoot{},
			}

			extension = &extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-mirror",
					Namespace: cluster.ObjectMeta.Name,
				},
				Spec: extensionsv1alpha1.ExtensionSpec{
					DefaultSpec: extensionsv1alpha1.DefaultSpec{
						ProviderConfig: &runtime.RawExtension{
							Object: &v1alpha1.MirrorConfig{
								TypeMeta: metav1.TypeMeta{
									APIVersion: v1alpha1.SchemeGroupVersion.String(),
									Kind:       "MirrorConfig",
								},
								Mirrors: []v1alpha1.MirrorConfiguration{
									{
										Upstream: "docker.io",
										Hosts: []v1alpha1.MirrorHost{
											{
												Host:         "https://mirror.gcr.io",
												Capabilities: []v1alpha1.MirrorHostCapability{v1alpha1.MirrorHostCapabilityPull, v1alpha1.MirrorHostCapabilityResolve},
											},
										},
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
									URL:          "https://mirror.foo.io",
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
					Namespace: "shoot--foo--bar",
				},
			}
			gctx := extensionscontextwebhook.NewGardenContext(fakeClient, osc)

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to get the cluster resource: could not get cluster for namespace 'shoot--foo--bar'")))
		})

		It("should do nothing if the shoot has a deletion timestamp set", func() {
			deletionTimestamp := metav1.Now()
			cluster.Shoot.DeletionTimestamp = &deletionTimestamp

			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)
			expectedContainerd := criConfig.Containerd.DeepCopy()

			Expect(ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)).To(Succeed())
			Expect(criConfig.Containerd).To(Equal(expectedContainerd))
		})

		It("should return err when it fails to get the extension", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to get extension 'shoot--foo--bar/registry-mirror'")))
		})

		It("should return err when extension .spec.providerConfig is nil", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)
			extension.Spec.ProviderConfig = nil

			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("extension 'shoot--foo--bar/registry-mirror' does not have a .spec.providerConfig specified")))
		})

		It("should return err when extension .spec.providerConfig cannot be decoded", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)
			extension.Spec.ProviderConfig = &runtime.RawExtension{Object: &corev1.Pod{}}

			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to decode providerConfig of extension 'shoot--foo--bar/registry-mirror'")))
		})

		It("should add additional registry config to a nil containerd registry configs", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)
			criConfig.Containerd = nil

			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)

			expectedRegistries := []extensionsv1alpha1.RegistryConfig{{
				Upstream: "docker.io",
				Server:   ptr.To("https://registry-1.docker.io"),
				Hosts: []extensionsv1alpha1.RegistryHost{
					{
						URL:          "https://mirror.gcr.io",
						Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability, extensionsv1alpha1.ResolveCapability},
					},
				},
			}}

			Expect(ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)).To(Succeed())
			Expect(criConfig.Containerd.Registries).To(ConsistOf(expectedRegistries))
		})

		It("should add additional registry config to the current containerd registry configs", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)

			expectedRegistries := criConfig.Containerd.DeepCopy().Registries
			expectedRegistries = append(expectedRegistries, extensionsv1alpha1.RegistryConfig{
				Upstream: "docker.io",
				Server:   ptr.To("https://registry-1.docker.io"),
				Hosts: []extensionsv1alpha1.RegistryHost{
					{
						URL:          "https://mirror.gcr.io",
						Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability, extensionsv1alpha1.ResolveCapability},
					},
				},
			})
			Expect(ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)).To(Succeed())
			Expect(criConfig.Containerd.Registries).To(ConsistOf(expectedRegistries))
		})

		It("should update existing registry config from containerd registry configs", func() {
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)

			expectedRegistries := criConfig.Containerd.DeepCopy().Registries
			expectedRegistries = append(expectedRegistries, extensionsv1alpha1.RegistryConfig{
				Upstream: "docker.io",
				Server:   ptr.To("https://registry-1.docker.io"),
				Hosts: []extensionsv1alpha1.RegistryHost{
					{
						URL:          "https://mirror.gcr.io",
						Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability, extensionsv1alpha1.ResolveCapability},
					},
				},
			})

			criConfig.Containerd.Registries = append(criConfig.Containerd.Registries, extensionsv1alpha1.RegistryConfig{
				Upstream: "docker.io",
				Server:   ptr.To("foo"),
				Hosts: []extensionsv1alpha1.RegistryHost{
					{
						URL:          "bar",
						Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PushCapability},
					},
				},
			})

			Expect(ensurer.EnsureCRIConfig(ctx, gctx, &criConfig, nil)).To(Succeed())
			Expect(criConfig.Containerd.Registries).To(ConsistOf(expectedRegistries))
		})
	})
})
