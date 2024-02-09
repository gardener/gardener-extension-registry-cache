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

package cache_test

import (
	"context"
	_ "embed"
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
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha2"
	"github.com/gardener/gardener-extension-registry-cache/pkg/webhook/cache"
)

func TestRegistryCacheWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registry Cache Webhook Suite")
}

var (
	//go:embed scripts/configure-containerd-registries.sh
	configureContainerdRegistriesScript string
)

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
		registryinstall.Install(scheme)

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

		It("should add additional files to the current ones", func() {
			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			Expect(ensurer.EnsureAdditionalFiles(ctx, nil, &files, nil)).To(Succeed())
			Expect(files).To(ConsistOf(oldFile, configureContainerdRegistriesFile(configureContainerdRegistriesScript)))
		})

		It("should overwrite existing files of the current ones", func() {
			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			files = append(files, configureContainerdRegistriesFile("echo 'foo'"))

			Expect(ensurer.EnsureAdditionalFiles(ctx, nil, &files, nil)).To(Succeed())
			Expect(files).To(ConsistOf(oldFile, configureContainerdRegistriesFile(configureContainerdRegistriesScript)))
		})
	})

	Describe("#EnsureAdditionalUnits", func() {
		var (
			oldUnit = extensionsv1alpha1.Unit{
				Name: "foo.service",
			}
			units []extensionsv1alpha1.Unit
		)

		BeforeEach(func() {
			units = []extensionsv1alpha1.Unit{oldUnit}
		})

		It("should return err when it fails to get the cluster", func() {
			osc := &extensionsv1alpha1.OperatingSystemConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "shoot--foo--bar",
				},
			}
			gctx := extensionscontextwebhook.NewGardenContext(fakeClient, osc)

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalUnits(ctx, gctx, &units, nil)
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

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			Expect(ensurer.EnsureAdditionalUnits(ctx, gctx, &units, nil)).To(Succeed())
			Expect(units).To(ConsistOf(oldUnit))
		})

		It("should do nothing if hibernation is enabled for Shoot", func() {
			cluster := &extensions.Cluster{
				Shoot: &gardencorev1beta1.Shoot{
					Spec: gardencorev1beta1.ShootSpec{
						Hibernation: &gardencorev1beta1.Hibernation{
							Enabled: ptr.To(true),
						},
					},
				},
			}
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			Expect(ensurer.EnsureAdditionalUnits(ctx, gctx, &units, nil)).To(Succeed())
			Expect(units).To(ConsistOf(oldUnit))
		})

		It("return err when it fails to get the extension", func() {
			cluster := &extensions.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "shoot--foo--bar"},
				Shoot:      &gardencorev1beta1.Shoot{},
			}
			gctx := extensionscontextwebhook.NewInternalGardenContext(cluster)

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalUnits(ctx, gctx, &units, nil)
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

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalUnits(ctx, gctx, &units, nil)
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

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			err := ensurer.EnsureAdditionalUnits(ctx, gctx, &units, nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to decode providerStatus of extension 'shoot--foo--bar/registry-cache'")))
		})

		It("should add additional unit to the current ones", func() {
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
							Object: &v1alpha2.RegistryStatus{
								TypeMeta: metav1.TypeMeta{
									APIVersion: v1alpha2.SchemeGroupVersion.String(),
									Kind:       "RegistryStatus",
								},
								Caches: []v1alpha2.RegistryCacheStatus{
									{
										Upstream: "docker.io",
										Endpoint: "http://10.0.0.1:5000",
									},
									{
										Upstream: "europe-docker.pkg.dev",
										Endpoint: "http://10.0.0.2:5000",
									},
								},
							},
						},
					},
				},
			}
			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			Expect(ensurer.EnsureAdditionalUnits(ctx, gctx, &units, nil)).To(Succeed())
			Expect(units).To(ConsistOf(oldUnit,
				configureContainerdRegistriesUnit("docker.io,http://10.0.0.1:5000,https://registry-1.docker.io europe-docker.pkg.dev,http://10.0.0.2:5000,https://europe-docker.pkg.dev"),
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
					Name:      "registry-cache",
					Namespace: cluster.ObjectMeta.Name,
				},
				Status: extensionsv1alpha1.ExtensionStatus{
					DefaultStatus: extensionsv1alpha1.DefaultStatus{
						ProviderStatus: &runtime.RawExtension{
							Object: &v1alpha2.RegistryStatus{
								TypeMeta: metav1.TypeMeta{
									APIVersion: v1alpha2.SchemeGroupVersion.String(),
									Kind:       "RegistryStatus",
								},
								Caches: []v1alpha2.RegistryCacheStatus{
									{
										Upstream: "docker.io",
										Endpoint: "http://10.0.0.1:5000",
									},
									{
										Upstream: "europe-docker.pkg.dev",
										Endpoint: "http://10.0.0.2:5000",
									},
								},
							},
						},
					},
				},
			}
			Expect(fakeClient.Create(ctx, extension)).To(Succeed())

			ensurer := cache.NewEnsurer(fakeClient, decoder, logger)

			units = append(units,
				configureContainerdRegistriesUnit("docker.io,foo,bar"),
			)

			Expect(ensurer.EnsureAdditionalUnits(ctx, gctx, &units, nil)).To(Succeed())
			Expect(units).To(ConsistOf(oldUnit,
				configureContainerdRegistriesUnit("docker.io,http://10.0.0.1:5000,https://registry-1.docker.io europe-docker.pkg.dev,http://10.0.0.2:5000,https://europe-docker.pkg.dev"),
			))
		})
	})
})

func configureContainerdRegistriesFile(script string) extensionsv1alpha1.File {
	return extensionsv1alpha1.File{
		Path:        "/opt/bin/configure-containerd-registries.sh",
		Permissions: ptr.To(int32(0744)),
		Content: extensionsv1alpha1.FileContent{
			Inline: &extensionsv1alpha1.FileContentInline{
				Encoding: "b64",
				Data:     base64.StdEncoding.EncodeToString([]byte(script)),
			},
		},
	}
}

func configureContainerdRegistriesUnit(args string) extensionsv1alpha1.Unit {
	return extensionsv1alpha1.Unit{
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
ExecStart=/opt/bin/configure-containerd-registries.sh ` + args),
	}
}
