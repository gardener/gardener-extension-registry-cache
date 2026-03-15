// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"encoding/json"
	"testing"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gardener/gardener-extension-registry-cache/pkg/admission/validator/cache"
	registryapi "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
)

func TestRegistryCacheValidator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registry Cache Validator Suite")
}

var _ = Describe("Shoot validator", func() {

	Describe("#Validate", func() {
		var (
			ctx  = context.Background()
			size = resource.MustParse("20Gi")

			decoder        runtime.Decoder
			shootValidator extensionswebhook.Validator

			shoot *core.Shoot
		)

		BeforeEach(func() {
			scheme := runtime.NewScheme()
			Expect(registryapi.AddToScheme(scheme)).To(Succeed())
			Expect(v1alpha3.AddToScheme(scheme)).To(Succeed())

			decoder = serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()

			shootValidator = cache.NewShootValidator(nil, decoder)

			shoot = &core.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "crazy-botany",
					Namespace: "garden-dev",
				},
				Spec: core.ShootSpec{
					Extensions: []core.Extension{
						{
							Type: "registry-cache",
							ProviderConfig: &runtime.RawExtension{
								Raw: encode(&v1alpha3.RegistryConfig{
									TypeMeta: metav1.TypeMeta{
										APIVersion: v1alpha3.SchemeGroupVersion.String(),
										Kind:       "RegistryConfig",
									},
									Caches: []v1alpha3.RegistryCache{
										{
											Upstream: "docker.io",
											Volume: &v1alpha3.Volume{
												Size: &size,
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

		Context("Shoot creation (old is nil)", func() {
			It("should return err when new is not a Shoot", func() {
				err := shootValidator.Validate(ctx, &corev1.Pod{}, nil)
				Expect(err).To(MatchError("wrong object type *v1.Pod"))
			})

			It("should do nothing when the Shoot does no specify a registry-cache extension", func() {
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
				Expect(err).To(MatchError("container runtime needs to be containerd when the registry-cache extension is enabled"))
			})

			It("should return err when registry-cache providerConfig is nil", func() {
				shoot.Spec.Extensions[0].ProviderConfig = nil

				err := shootValidator.Validate(ctx, shoot, nil)
				Expect(err).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("spec.extensions[0].providerConfig"),
					"Detail": Equal("providerConfig is required for the registry-cache extension"),
				})))
			})

			It("should return err when registry-cache providerConfig cannot be decoded", func() {
				shoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{
					Raw: []byte(`{"bar": "baz"}`),
				}

				err := shootValidator.Validate(ctx, shoot, nil)
				Expect(err).To(MatchError(ContainSubstring("failed to decode providerConfig")))
			})

			It("should return err when registry-cache providerConfig is invalid", func() {
				shoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{
					Raw: encode(&v1alpha3.RegistryConfig{
						TypeMeta: metav1.TypeMeta{
							APIVersion: v1alpha3.SchemeGroupVersion.String(),
							Kind:       "RegistryConfig",
						},
						Caches: []v1alpha3.RegistryCache{
							{
								Upstream: "https://registry.example.com",
							},
						},
					}),
				}

				err := shootValidator.Validate(ctx, shoot, nil)
				Expect(err).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("spec.extensions[0].providerConfig.caches[0].upstream"),
					"BadValue": Equal("https://registry.example.com"),
				}))))
			})

			It("should succeed for valid Shoot", func() {
				Expect(shootValidator.Validate(ctx, shoot, nil)).To(Succeed())
			})
		})

		Context("Shoot update (old is set)", func() {
			var oldShoot *core.Shoot

			BeforeEach(func() {
				oldShoot = shoot.DeepCopy()
			})

			It("should return err when old is not a Shoot", func() {
				err := shootValidator.Validate(ctx, shoot, &corev1.Pod{})
				Expect(err).To(MatchError("wrong object type *v1.Pod for old object"))
			})

			It("should return err when old Shoot registry-cache providerConfig is nil", func() {
				oldShoot.Spec.Extensions[0].ProviderConfig = nil

				err := shootValidator.Validate(ctx, shoot, oldShoot)
				Expect(err).To(MatchError(ContainSubstring("providerConfig is not available on old Shoot")))
			})

			It("should return err when old Shoot registry-cache providerConfig cannot be decoded", func() {
				oldShoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{
					Raw: []byte(`{"bar": "baz"}`),
				}

				err := shootValidator.Validate(ctx, shoot, oldShoot)
				Expect(err).To(MatchError(ContainSubstring("failed to decode providerConfig")))
			})

			It("should return err when registry-cache providerConfig update is invalid", func() {
				newSize := resource.MustParse("42Gi")
				shoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{
					Raw: encode(&v1alpha3.RegistryConfig{
						TypeMeta: metav1.TypeMeta{
							APIVersion: v1alpha3.SchemeGroupVersion.String(),
							Kind:       "RegistryConfig",
						},
						Caches: []v1alpha3.RegistryCache{
							{
								Upstream: "docker.io",
								Volume: &v1alpha3.Volume{
									Size: &newSize,
								},
							},
						},
					}),
				}

				err := shootValidator.Validate(ctx, shoot, oldShoot)
				Expect(err).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("spec.extensions[0].providerConfig.caches[0].volume.size"),
					"BadValue": Equal("42Gi"),
					"Detail":   Equal("field is immutable"),
				}))))
			})

			It("should exit earlier when no semantic change in providerConfig is detected", func() {
				shoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{
					Raw: encode(&v1alpha3.RegistryConfig{
						TypeMeta: metav1.TypeMeta{
							APIVersion: v1alpha3.SchemeGroupVersion.String(),
							Kind:       "RegistryConfig",
						},
						Caches: []v1alpha3.RegistryCache{
							{
								Upstream: "https://registry.example.com", // invalid upstream
							},
						},
					}),
				}
				oldShoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{
					Raw: encode(&v1alpha3.RegistryConfig{
						TypeMeta: metav1.TypeMeta{
							APIVersion: v1alpha3.SchemeGroupVersion.String(),
							Kind:       "RegistryConfig",
						},
						Caches: []v1alpha3.RegistryCache{
							{
								Upstream: "https://registry.example.com", // invalid upstream
								GarbageCollection: &v1alpha3.GarbageCollection{
									TTL: v1alpha3.DefaultTTL,
								},
							},
						},
					}),
				}
				Expect(shootValidator.Validate(ctx, shoot, oldShoot)).To(Succeed())
			})
		})

		Context("Upstream credentials", func() {
			var (
				fakeClient client.Client

				secret *corev1.Secret
			)

			BeforeEach(func() {
				fakeClient = fakeclient.NewClientBuilder().Build()
				shootValidator = cache.NewShootValidator(fakeClient, decoder)

				secret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "docker-creds-v1",
						Namespace: "garden-dev",
					},
					Immutable: ptr.To(true),
					Data: map[string][]byte{
						"username": []byte("john"),
						"password": []byte("swordfish"),
					},
				}
				shoot.Spec.Resources = []core.NamedResourceReference{
					{
						Name: "docker-creds",
						ResourceRef: autoscalingv1.CrossVersionObjectReference{
							Kind: "Secret",
							Name: "docker-creds-v1",
						},
					},
				}
				shoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{
					Raw: encode(&v1alpha3.RegistryConfig{
						TypeMeta: metav1.TypeMeta{
							APIVersion: v1alpha3.SchemeGroupVersion.String(),
							Kind:       "RegistryConfig",
						},
						Caches: []v1alpha3.RegistryCache{
							{
								Upstream: "docker.io",
								Volume: &v1alpha3.Volume{
									Size: &size,
								},
								SecretReferenceName: ptr.To("docker-creds"),
							},
						},
					}),
				}
			})

			It("should succeed for valid secret reference", func() {
				Expect(fakeClient.Create(ctx, secret)).To(Succeed())
				Expect(shootValidator.Validate(ctx, shoot, nil)).To(Succeed())
			})

			DescribeTable("it should fail",
				func(namedRefs []core.NamedResourceReference) {
					shoot.Spec.Resources = namedRefs

					Expect(shootValidator.Validate(ctx, shoot, nil)).To(ConsistOf(
						PointTo(MatchFields(IgnoreExtras, Fields{
							"Type":   Equal(field.ErrorTypeInvalid),
							"Field":  Equal("spec.extensions[0].providerConfig.caches[0].secretReferenceName"),
							"Detail": ContainSubstring("failed to find referenced resource with name docker-creds and kind Secret"),
						})),
					))
				},
				Entry("when reference is missing", []core.NamedResourceReference{}),
				Entry("when reference has wrong kind", []core.NamedResourceReference{
					{
						Name: "docker-creds",
						ResourceRef: autoscalingv1.CrossVersionObjectReference{
							Kind: "ConfigMap",
							Name: "docker-creds-v1",
						},
					},
				}),
			)

			It("should return err when failed to get secret ", func() {
				Expect(shootValidator.Validate(ctx, shoot, nil)).To(MatchError(`failed to get secret garden-dev/docker-creds-v1 for secretReferenceName docker-creds: secrets "docker-creds-v1" not found`))
			})

			It("should return err when secret is invalid", func() {
				secret.Immutable = ptr.To(false)
				delete(secret.Data, "password")
				Expect(fakeClient.Create(ctx, secret)).To(Succeed())

				Expect(shootValidator.Validate(ctx, shoot, nil)).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("spec.extensions[0].providerConfig.caches[0].secretReferenceName"),
						"Detail": ContainSubstring(`the referenced secret "garden-dev/docker-creds-v1" should be immutable`),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("spec.extensions[0].providerConfig.caches[0].secretReferenceName"),
						"Detail": ContainSubstring(`the referenced secret "garden-dev/docker-creds-v1" should have only two data entries`),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("spec.extensions[0].providerConfig.caches[0].secretReferenceName"),
						"Detail": ContainSubstring(`missing "password" data entry in the referenced secret "garden-dev/docker-creds-v1"`),
					})),
				))
			})
		})
	})
})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}
