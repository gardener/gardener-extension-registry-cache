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

package mirror_test

import (
	"context"
	"fmt"
	"path/filepath"
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
	"k8s.io/utils/pointer"
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

	Describe("#EnsureAdditionalFiles", func() {
		var (
			oldFile = extensionsv1alpha1.File{
				Path: "/var/lib/foo.sh",
			}
			files []extensionsv1alpha1.File
		)

		BeforeEach(func() {
			files = []extensionsv1alpha1.File{oldFile}
		})

		It("should return err when it fails to get the cluster", func() {
			osc := &extensionsv1alpha1.OperatingSystemConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "shoot--foo--bar",
				},
			}
			gctx := extensionscontextwebhook.NewGardenContext(fakeClient, osc)

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to get the cluster resource: could not get cluster for namespace 'shoot--foo--bar'")))
		})

		It("should do nothing if the shoot has a deletion timestamp set", func() {
			deletionTimestamp := metav1.Now()
			cluster := &extensions.Cluster{
				Shoot: &gardencorev1beta1.Shoot{
					ObjectMeta: metav1.ObjectMeta{
						DeletionTimestamp: &deletionTimestamp,
					},
				},
			}
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)

			Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
			Expect(files).To(ConsistOf(oldFile))
		})

		It("return err when it fails to get the extesion", func() {
			cluster := &extensions.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "shoot--foo--bar"},
				Shoot:      &gardencorev1beta1.Shoot{},
			}
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to get extension 'shoot--foo--bar/registry-mirror'")))
		})

		It("should return err when extension .spec.providerConfig is nil", func() {
			cluster := &extensions.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "shoot--foo--bar"},
				Shoot:      &gardencorev1beta1.Shoot{},
			}
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			extension := &extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-mirror",
					Namespace: cluster.ObjectMeta.Name,
				},
				Spec: extensionsv1alpha1.ExtensionSpec{
					DefaultSpec: extensionsv1alpha1.DefaultSpec{
						ProviderConfig: nil,
					},
				},
			}
			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("extension 'shoot--foo--bar/registry-mirror' does not have a .spec.providerConfig specified")))
		})

		It("should return err when extension .spec.providerConfig cannot be decoded", func() {
			cluster := &extensions.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "shoot--foo--bar"},
				Shoot:      &gardencorev1beta1.Shoot{},
			}
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			extension := &extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-mirror",
					Namespace: cluster.ObjectMeta.Name,
				},
				Spec: extensionsv1alpha1.ExtensionSpec{
					DefaultSpec: extensionsv1alpha1.DefaultSpec{
						ProviderConfig: &runtime.RawExtension{
							Object: &corev1.Pod{},
						},
					},
				},
			}
			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to decode providerConfig of extension 'shoot--foo--bar/registry-mirror'")))
		})

		It("should add additional file to the current ones", func() {
			cluster := &extensions.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "shoot--foo--bar"},
				Shoot:      &gardencorev1beta1.Shoot{},
			}
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			extension := &extensionsv1alpha1.Extension{
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
			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)

			Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
			Expect(files).To(ConsistOf(oldFile,
				hostsTOMLFile("docker.io", "https://registry-1.docker.io", "https://mirror.gcr.io", `["pull", "resolve"]`),
			))
		})

		It("should overwrite existing unit of the current ones", func() {
			cluster := &extensions.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "shoot--foo--bar"},
				Shoot:      &gardencorev1beta1.Shoot{},
			}
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			extension := &extensionsv1alpha1.Extension{
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
			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := mirror.NewEnsurer(fakeClient, decoder, logger)

			files = append(files,
				hostsTOMLFile("docker.io", "foo", "bar", "baz"),
			)

			Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
			Expect(files).To(ConsistOf(oldFile,
				hostsTOMLFile("docker.io", "https://registry-1.docker.io", "https://mirror.gcr.io", `["pull", "resolve"]`),
			))
		})
	})
})

func hostsTOMLFile(upstream, upstreamServer, mirrorHost, capabilities string) extensionsv1alpha1.File {
	return extensionsv1alpha1.File{
		Path:        filepath.Join("/etc/containerd/certs.d/", upstream, "hosts.toml"),
		Permissions: pointer.Int32(0644),
		Content: extensionsv1alpha1.FileContent{
			Inline: &extensionsv1alpha1.FileContentInline{
				Data: fmt.Sprintf(`server = "%s"

[host."%s"]
  capabilities = %s
`, upstreamServer, mirrorHost, capabilities),
			},
		},
	}
}
