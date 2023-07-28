// Copyright (c) 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package operatingsystemconfig_test

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

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
	"github.com/gardener/gardener-extension-registry-cache/pkg/webhook/operatingsystemconfig"
)

func TestOperatingSystemConfigWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OperatingSystemConfig Webhook Suite")
}

var _ = Describe("Ensurer", func() {
	var (
		logger = logr.Discard()
		ctx    = context.TODO()

		decoder    runtime.Decoder
		fakeClient client.Client
	)

	BeforeEach(func() {
		scheme := runtime.NewScheme()
		Expect(extensionsv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())

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

			ensurer := operatingsystemconfig.NewEnsurer(fakeClient, decoder, logger)

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

			ensurer := operatingsystemconfig.NewEnsurer(fakeClient, decoder, logger)

			Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
			Expect(files).To(ConsistOf(oldFile))
		})

		It("should do nothing if hibernation is enabled for Shoot", func() {
			cluster := &extensions.Cluster{
				Shoot: &gardencorev1beta1.Shoot{
					Spec: gardencorev1beta1.ShootSpec{
						Hibernation: &gardencorev1beta1.Hibernation{
							Enabled: pointer.Bool(true),
						},
					},
				},
			}
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			ensurer := operatingsystemconfig.NewEnsurer(fakeClient, decoder, logger)

			Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
			Expect(files).To(ConsistOf(oldFile))
		})

		It("return err when it fails to get the extesion", func() {
			cluster := &extensions.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "shoot--foo--bar"},
				Shoot:      &gardencorev1beta1.Shoot{},
			}
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			ensurer := operatingsystemconfig.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to get extension 'shoot--foo--bar/registry-cache'")))
		})

		It("should return err when extension .status.providerStatus is nil", func() {
			cluster := &extensions.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "shoot--foo--bar"},
				Shoot:      &gardencorev1beta1.Shoot{},
			}
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			extension := &extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-cache",
					Namespace: cluster.ObjectMeta.Name,
				},
				Status: extensionsv1alpha1.ExtensionStatus{
					DefaultStatus: extensionsv1alpha1.DefaultStatus{
						ProviderStatus: nil,
					},
				},
			}
			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := operatingsystemconfig.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("extension 'shoot--foo--bar/registry-cache' does not have a .status.providerStatus specified")))
		})

		It("should return err when extension .status.providerStatus cannot be decoded", func() {
			cluster := &extensions.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "shoot--foo--bar"},
				Shoot:      &gardencorev1beta1.Shoot{},
			}
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			extension := &extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-cache",
					Namespace: cluster.ObjectMeta.Name,
				},
				Status: extensionsv1alpha1.ExtensionStatus{
					DefaultStatus: extensionsv1alpha1.DefaultStatus{
						ProviderStatus: &runtime.RawExtension{
							Object: &corev1.Pod{},
						},
					},
				},
			}
			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := operatingsystemconfig.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to decode providerStatus of extension 'shoot--foo--bar/registry-cache'")))
		})

		It("should add additional files to the current ones", func() {
			cluster := &extensions.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "shoot--foo--bar"},
				Shoot:      &gardencorev1beta1.Shoot{},
			}
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			extension := &extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-cache",
					Namespace: cluster.ObjectMeta.Name,
				},
				Status: extensionsv1alpha1.ExtensionStatus{
					DefaultStatus: extensionsv1alpha1.DefaultStatus{
						ProviderStatus: &runtime.RawExtension{
							Object: &v1alpha1.RegistryStatus{
								Caches: []v1alpha1.RegistryCacheStatus{
									{
										Upstream: "docker.io",
										Endpoint: "http://10.0.0.1:5000",
									},
									{
										Upstream: "eu.gcr.io",
										Endpoint: "http://10.0.0.2:5000",
									},
								},
							},
						},
					},
				},
			}
			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := operatingsystemconfig.NewEnsurer(fakeClient, decoder, logger)

			Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
			Expect(files).To(ConsistOf(oldFile,
				hostsTOMLFile("docker.io", "registry-1.docker.io", "http://10.0.0.1:5000"),
				hostsTOMLFile("eu.gcr.io", "eu.gcr.io", "http://10.0.0.2:5000"),
			))
		})

		It("should overwrite existing files of the current ones", func() {
			cluster := &extensions.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "shoot--foo--bar"},
				Shoot:      &gardencorev1beta1.Shoot{},
			}
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			extension := &extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-cache",
					Namespace: cluster.ObjectMeta.Name,
				},
				Status: extensionsv1alpha1.ExtensionStatus{
					DefaultStatus: extensionsv1alpha1.DefaultStatus{
						ProviderStatus: &runtime.RawExtension{
							Object: &v1alpha1.RegistryStatus{
								Caches: []v1alpha1.RegistryCacheStatus{
									{
										Upstream: "docker.io",
										Endpoint: "http://10.0.0.1:5000",
									},
									{
										Upstream: "eu.gcr.io",
										Endpoint: "http://10.0.0.2:5000",
									},
								},
							},
						},
					},
				},
			}
			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := operatingsystemconfig.NewEnsurer(fakeClient, decoder, logger)

			files = append(files,
				hostsTOMLFile("docker.io", "foo", "bar"),
				hostsTOMLFile("eu.gcr.io", "baz", "bazz"),
			)

			Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
			Expect(files).To(ConsistOf(oldFile,
				hostsTOMLFile("docker.io", "registry-1.docker.io", "http://10.0.0.1:5000"),
				hostsTOMLFile("eu.gcr.io", "eu.gcr.io", "http://10.0.0.2:5000"),
			))
		})
	})
})

func hostsTOMLFile(upstream, upstreamServer, cacheEndpoint string) extensionsv1alpha1.File {
	return extensionsv1alpha1.File{
		Path:        filepath.Join("/etc/containerd/certs.d/", upstream, "hosts.toml"),
		Permissions: pointer.Int32(0644),
		Content: extensionsv1alpha1.FileContent{
			Inline: &extensionsv1alpha1.FileContentInline{
				Data: fmt.Sprintf(`server = "%s"

[host."%s"]
  capabilities = ["pull", "resolve"]
`, upstreamServer, cacheEndpoint),
			},
		},
	}
}
