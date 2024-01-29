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
	"encoding/json"
	"testing"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener-extension-registry-cache/pkg/admission/validator/mirror"
	mirrorinstall "github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror/install"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror/v1alpha1"
	registryinstall "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/install"
	registryv1alpha1 "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
)

func TestRegistryMirrorValidator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registry Mirror Validator Suite")
}

var _ = Describe("Shoot validator", func() {

	var (
		ctx  = context.Background()
		size = resource.MustParse("20Gi")

		shootValidator extensionswebhook.Validator

		shoot *core.Shoot
	)

	Describe("#Validate", func() {
		BeforeEach(func() {
			scheme := runtime.NewScheme()
			mirrorinstall.Install(scheme)
			registryinstall.Install(scheme)

			decoder := serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()

			shootValidator = mirror.NewShootValidator(decoder)

			shoot = &core.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "garden-tst",
					Name:      "tst",
				},
				Spec: core.ShootSpec{
					Extensions: []core.Extension{
						{
							Type: "registry-mirror",
							ProviderConfig: &runtime.RawExtension{
								Raw: encode(&v1alpha1.MirrorConfig{
									TypeMeta: metav1.TypeMeta{
										APIVersion: v1alpha1.SchemeGroupVersion.String(),
										Kind:       "MirrorConfig",
									},
									Mirrors: []v1alpha1.MirrorConfiguration{
										{
											Upstream: "docker.io",
											Hosts: []v1alpha1.MirrorHost{
												{
													Host: "https://mirror.gcr.io",
												},
											},
										},
									},
								}),
							},
						},
					},
					Provider: core.Provider{
						Workers: []core.Worker{
							{
								CRI: &core.CRI{Name: "containerd"},
							},
						},
					},
				},
			}
		})

		It("should return err when new is not a Shoot", func() {
			err := shootValidator.Validate(ctx, &corev1.Pod{}, nil)
			Expect(err).To(MatchError("wrong object type *v1.Pod"))
		})

		It("should do nothing when the Shoot does no specify a registry-mirror extension", func() {
			shoot.Spec.Extensions[0].Type = "foo"

			Expect(shootValidator.Validate(ctx, shoot, nil)).To(Succeed())
		})

		It("should return err when there is container runtime that is not containerd", func() {
			worker := core.Worker{
				CRI: &core.CRI{
					Name: "docker",
				},
			}
			shoot.Spec.Provider.Workers = append(shoot.Spec.Provider.Workers, worker)

			err := shootValidator.Validate(ctx, shoot, nil)
			Expect(err).To(MatchError("container runtime needs to be containerd when the registry-mirror extension is enabled"))
		})

		It("should return err when registry-mirror providerConfig is nil", func() {
			shoot.Spec.Extensions[0].ProviderConfig = nil

			err := shootValidator.Validate(ctx, shoot, nil)
			Expect(err).To(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("spec.extensions[0].providerConfig"),
				"Detail": Equal("providerConfig is required for the registry-mirror extension"),
			})))
		})

		It("should return err when registry-mirror providerConfig cannot be decoded", func() {
			shoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{
				Raw: []byte(`{"bar": "baz"}`),
			}

			err := shootValidator.Validate(ctx, shoot, nil)
			Expect(err).To(MatchError(ContainSubstring("failed to decode providerConfig")))
		})

		It("should return err when registry-mirror providerConfig is invalid", func() {
			shoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{
				Raw: encode(&v1alpha1.MirrorConfig{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha1.SchemeGroupVersion.String(),
						Kind:       "MirrorConfig",
					},
					Mirrors: []v1alpha1.MirrorConfiguration{
						{
							Upstream: "registry.example.com",
							Hosts:    nil,
						},
					},
				}),
			}

			err := shootValidator.Validate(ctx, shoot, nil)
			Expect(err).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("spec.extensions[0].providerConfig.mirrors[0].hosts"),
				"Detail": Equal("at least one host must be provided"),
			}))))
		})

		It("should return err when registry-cache providerConfig is nil", func() {
			shoot.Spec.Extensions = append(shoot.Spec.Extensions, core.Extension{
				Type:           "registry-cache",
				ProviderConfig: nil,
			})

			err := shootValidator.Validate(ctx, shoot, nil)
			Expect(err).To(MatchError(ContainSubstring("providerConfig is not available for registry-cache extension")))
		})

		It("should return err when registry-cache providerConfig cannot be decoded", func() {
			shoot.Spec.Extensions = append(shoot.Spec.Extensions, core.Extension{
				Type: "registry-cache",
				ProviderConfig: &runtime.RawExtension{
					Raw: []byte(`{"bar": "baz"}`),
				},
			})

			err := shootValidator.Validate(ctx, shoot, nil)
			Expect(err).To(MatchError(ContainSubstring("failed to decode providerConfig")))
		})

		It("should return err when registry-mirror providerConfig is invalid against registry-cache providerConfig", func() {
			shoot.Spec.Extensions = append(shoot.Spec.Extensions, core.Extension{
				Type: "registry-cache",
				ProviderConfig: &runtime.RawExtension{
					Raw: encode(&registryv1alpha1.RegistryConfig{
						TypeMeta: metav1.TypeMeta{
							APIVersion: registryv1alpha1.SchemeGroupVersion.String(),
							Kind:       "RegistryConfig",
						},
						Caches: []registryv1alpha1.RegistryCache{
							{
								Upstream: "docker.io",
								Size:     &size,
							},
						},
					}),
				},
			})

			err := shootValidator.Validate(ctx, shoot, nil)
			Expect(err).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":     Equal(field.ErrorTypeDuplicate),
				"Field":    Equal("spec.extensions[0].providerConfig.mirrors[0].upstream"),
				"BadValue": Equal("docker.io"),
			}))))
		})

		It("should succeed for valid Shoot", func() {
			Expect(shootValidator.Validate(ctx, shoot, nil)).To(Succeed())
		})
	})
})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}
