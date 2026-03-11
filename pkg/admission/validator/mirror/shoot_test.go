// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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

	"github.com/gardener/gardener-extension-registry-cache/pkg/admission/validator/mirror"
	mirrorinstall "github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror/install"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror/v1alpha1"
	registryinstall "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/install"
	registryv1alpha3 "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
)

func TestRegistryMirrorValidator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registry Mirror Validator Suite")
}

var _ = Describe("Shoot validator", func() {

	var (
		ctx  = context.Background()
		size = resource.MustParse("20Gi")

		decoder        runtime.Decoder
		shootValidator extensionswebhook.Validator

		shoot *core.Shoot
	)

	Describe("#Validate", func() {
		BeforeEach(func() {
			scheme := runtime.NewScheme()
			mirrorinstall.Install(scheme)
			registryinstall.Install(scheme)

			decoder = serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()

			shootValidator = mirror.NewShootValidator(nil, decoder)

			shoot = &core.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "crazy-botany",
					Namespace: "garden-dev",
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
					Raw: encode(&registryv1alpha3.RegistryConfig{
						TypeMeta: metav1.TypeMeta{
							APIVersion: registryv1alpha3.SchemeGroupVersion.String(),
							Kind:       "RegistryConfig",
						},
						Caches: []registryv1alpha3.RegistryCache{
							{
								Upstream: "docker.io",
								Volume: &registryv1alpha3.Volume{
									Size: &size,
								},
							},
						},
					}),
				},
			})

			err := shootValidator.Validate(ctx, shoot, nil)
			Expect(err).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":     Equal(field.ErrorTypeInvalid),
				"Field":    Equal("spec.extensions[0].providerConfig.mirrors[0].upstream"),
				"BadValue": Equal("docker.io"),
				"Detail":   Equal("upstream host 'docker.io' is also configured as a registry cache upstream"),
			}))))
		})

		It("should succeed for valid Shoot", func() {
			Expect(shootValidator.Validate(ctx, shoot, nil)).To(Succeed())
		})

		Context("CA bundle secret reference", func() {
			const caBundle = "-----BEGIN CERTIFICATE-----\nMIICRzCCAfGgAwIBAgIJALMb7ecMIk3MMA0GCSqGSIb3DQEBCwUAMH4xCzAJBgNV\nBAYTAkdCMQ8wDQYDVQQIDAZMb25kb24xDzANBgNVBAcMBkxvbmRvbjEYMBYGA1UE\nCgwPR2xvYmFsIFNlY3VyaXR5MRYwFAYDVQQLDA1JVCBEZXBhcnRtZW50MRswGQYD\nVQQDDBJ0ZXN0LWNlcnRpZmljYXRlLTAwIBcNMTcwNDI2MjMyNjUyWhgPMjExNzA0\nMDIyMzI2NTJaMH4xCzAJBgNVBAYTAkdCMQ8wDQYDVQQIDAZMb25kb24xDzANBgNV\nBAcMBkxvbmRvbjEYMBYGA1UECgwPR2xvYmFsIFNlY3VyaXR5MRYwFAYDVQQLDA1J\nVCBEZXBhcnRtZW50MRswGQYDVQQDDBJ0ZXN0LWNlcnRpZmljYXRlLTAwXDANBgkq\nhkiG9w0BAQEFAANLADBIAkEAtBMa7NWpv3BVlKTCPGO/LEsguKqWHBtKzweMY2CV\ntAL1rQm913huhxF9w+ai76KQ3MHK5IVnLJjYYA5MzP2H5QIDAQABo1AwTjAdBgNV\nHQ4EFgQU22iy8aWkNSxv0nBxFxerfsvnZVMwHwYDVR0jBBgwFoAU22iy8aWkNSxv\n0nBxFxerfsvnZVMwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAANBAEOefGbV\nNcHxklaW06w6OBYJPwpIhCVozC1qdxGX1dg8VkEKzjOzjgqVD30m59OFmSlBmHsl\nnkVA6wyOSDYBf3o=\n-----END CERTIFICATE-----"

			var (
				fakeClient client.Client

				secret *corev1.Secret
			)

			BeforeEach(func() {
				fakeClient = fakeclient.NewClientBuilder().Build()
				shootValidator = mirror.NewShootValidator(fakeClient, decoder)

				secret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ca-bundle-v1",
						Namespace: "garden-dev",
					},
					Immutable: ptr.To(true),
					Data: map[string][]byte{
						"bundle.crt": []byte(caBundle),
					},
				}
				shoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{
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
										Host:                        "https://private-mirror.internal",
										CABundleSecretReferenceName: ptr.To("ca-bundle"),
									},
								},
							},
						},
					}),
				}
				shoot.Spec.Resources = []core.NamedResourceReference{
					{
						Name: "ca-bundle",
						ResourceRef: autoscalingv1.CrossVersionObjectReference{
							Kind: "Secret",
							Name: "ca-bundle-v1",
						},
					},
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
							"Field":  Equal("spec.extensions[0].providerConfig.mirrors[0].hosts[0].caBundleSecretReferenceName"),
							"Detail": ContainSubstring("failed to find referenced resource with name ca-bundle and kind Secret"),
						})),
					))
				},
				Entry("when reference is missing", []core.NamedResourceReference{}),
				Entry("when reference has wrong kind", []core.NamedResourceReference{
					{
						Name: "ca-bundle",
						ResourceRef: autoscalingv1.CrossVersionObjectReference{
							Kind: "ConfigMap",
							Name: "ca-bundle-v1",
						},
					},
				}),
			)

			It("should return err when failed to get secret ", func() {
				Expect(shootValidator.Validate(ctx, shoot, nil)).To(MatchError(`failed to get secret garden-dev/ca-bundle-v1 for caBundleSecretReferenceName ca-bundle: secrets "ca-bundle-v1" not found`))
			})

			It("should return err when secret is invalid", func() {
				secret.Immutable = ptr.To(false)
				delete(secret.Data, "bundle.crt")
				Expect(fakeClient.Create(ctx, secret)).To(Succeed())

				Expect(shootValidator.Validate(ctx, shoot, nil)).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("spec.extensions[0].providerConfig.mirrors[0].hosts[0].caBundleSecretReferenceName"),
						"Detail": ContainSubstring(`the referenced CA bundle secret "garden-dev/ca-bundle-v1" should be immutable`),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("spec.extensions[0].providerConfig.mirrors[0].hosts[0].caBundleSecretReferenceName"),
						"Detail": ContainSubstring(`missing "bundle.crt" data entry in the referenced CA bundle secret "garden-dev/ca-bundle-v1"`),
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
