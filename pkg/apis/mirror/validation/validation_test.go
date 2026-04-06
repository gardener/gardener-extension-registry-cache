// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	mirrorapi "github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror"
	. "github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror/validation"
)

var _ = Describe("Validation", func() {
	var (
		fldPath      *field.Path
		mirrorConfig *mirrorapi.MirrorConfig
	)

	BeforeEach(func() {
		fldPath = field.NewPath("providerConfig")
		mirrorConfig = &mirrorapi.MirrorConfig{
			Mirrors: []mirrorapi.MirrorConfiguration{
				{
					Upstream: "docker.io",
					Hosts: []mirrorapi.MirrorHost{
						{
							Host:         "https://mirror.gcr.io",
							Capabilities: []mirrorapi.MirrorHostCapability{mirrorapi.MirrorHostCapabilityPull, mirrorapi.MirrorHostCapabilityResolve},
						},
					},
				},
			},
		}
	})

	Describe("#ValidateMirrorConfig", func() {
		It("should allow valid configuration", func() {
			Expect(ValidateMirrorConfig(mirrorConfig, fldPath)).To(BeEmpty())
		})

		It("should allow valid configuration that has a path in host URL", func() {
			mirrorConfig.Mirrors[0].Hosts[0].Host = "https://mirror.gcr.io/v2/quay"
			Expect(ValidateMirrorConfig(mirrorConfig, fldPath)).To(BeEmpty())
		})

		It("should deny configuration without a mirror", func() {
			mirrorConfig = &mirrorapi.MirrorConfig{Mirrors: nil}
			Expect(ValidateMirrorConfig(mirrorConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("providerConfig.mirrors"),
					"Detail": ContainSubstring("at least one mirror must be provided"),
				})),
			))

			mirrorConfig = &mirrorapi.MirrorConfig{Mirrors: []mirrorapi.MirrorConfiguration{}}
			Expect(ValidateMirrorConfig(mirrorConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("providerConfig.mirrors"),
					"Detail": ContainSubstring("at least one mirror must be provided"),
				})),
			))
		})

		It("should deny invalid upstreams", func() {
			mirrorConfig.Mirrors[0].Upstream = ""

			mirrorConfig.Mirrors = append(mirrorConfig.Mirrors,
				mirrorapi.MirrorConfiguration{
					Upstream: "docker.io.",
					Hosts:    []mirrorapi.MirrorHost{{Host: "https://mirror.gcr.io"}},
				},
				mirrorapi.MirrorConfiguration{
					Upstream: ".docker.io",
					Hosts:    []mirrorapi.MirrorHost{{Host: "https://mirror.gcr.io"}},
				},
				mirrorapi.MirrorConfiguration{
					Upstream: "https://docker.io",
					Hosts:    []mirrorapi.MirrorHost{{Host: "https://mirror.gcr.io"}},
				},
				mirrorapi.MirrorConfiguration{
					Upstream: "docker.io:0443",
					Hosts:    []mirrorapi.MirrorHost{{Host: "https://mirror.gcr.io"}},
				},
			)

			Expect(ValidateMirrorConfig(mirrorConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.mirrors[0].upstream"),
					"BadValue": Equal(""),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.mirrors[1].upstream"),
					"BadValue": Equal("docker.io."),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.mirrors[2].upstream"),
					"BadValue": Equal(".docker.io"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.mirrors[3].upstream"),
					"BadValue": Equal("https://docker.io"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.mirrors[4].upstream"),
					"BadValue": Equal("docker.io:0443"),
				})),
			))
		})

		It("should deny configuration of mirror without a host", func() {
			mirrorConfig.Mirrors[0].Hosts = nil

			Expect(ValidateMirrorConfig(mirrorConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("providerConfig.mirrors[0].hosts"),
					"Detail": ContainSubstring("at least one host must be provided"),
				})),
			))
		})

		It("should deny mirror host without a scheme", func() {
			mirrorConfig = &mirrorapi.MirrorConfig{
				Mirrors: []mirrorapi.MirrorConfiguration{
					{
						Upstream: "docker.io",
						Hosts: []mirrorapi.MirrorHost{
							{Host: "public-mirror.example.com"},
							{Host: "docker-mirror.internal"},
						},
					},
				},
			}

			Expect(ValidateMirrorConfig(mirrorConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.mirrors[0].hosts[0].host"),
					"BadValue": Equal("public-mirror.example.com"),
					"Detail":   Equal("url must start with 'http://' or 'https://' scheme"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.mirrors[0].hosts[1].host"),
					"BadValue": Equal("docker-mirror.internal"),
					"Detail":   Equal("url must start with 'http://' or 'https://' scheme"),
				})),
			))
		})

		It("should deny duplicate mirror hosts", func() {
			mirrorConfig = &mirrorapi.MirrorConfig{
				Mirrors: []mirrorapi.MirrorConfiguration{
					{
						Upstream: "docker.io",
						Hosts: []mirrorapi.MirrorHost{
							{Host: "https://mirror.gcr.io"},
							{Host: "https://mirror.gcr.io"},
						},
					},
				},
			}

			Expect(ValidateMirrorConfig(mirrorConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("providerConfig.mirrors[0].hosts[1].host"),
				})),
			))
		})

		It("should deny invalid mirror host capability", func() {
			mirrorConfig = &mirrorapi.MirrorConfig{
				Mirrors: []mirrorapi.MirrorConfiguration{
					{
						Upstream: "docker.io",
						Hosts: []mirrorapi.MirrorHost{
							{
								Host:         "https://mirror.gcr.io",
								Capabilities: []mirrorapi.MirrorHostCapability{"foo"},
							},
						},
					},
				},
			}

			Expect(ValidateMirrorConfig(mirrorConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeNotSupported),
					"Field":    Equal("providerConfig.mirrors[0].hosts[0].capabilities"),
					"BadValue": Equal("foo"),
					"Detail":   Equal(`supported values: "pull", "resolve"`),
				})),
			))
		})

		It("should deny duplicate mirror host capability", func() {
			mirrorConfig = &mirrorapi.MirrorConfig{
				Mirrors: []mirrorapi.MirrorConfiguration{
					{
						Upstream: "docker.io",
						Hosts: []mirrorapi.MirrorHost{
							{
								Host:         "https://mirror.gcr.io",
								Capabilities: []mirrorapi.MirrorHostCapability{"pull", "resolve", "pull"},
							},
						},
					},
				},
			}

			Expect(ValidateMirrorConfig(mirrorConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeDuplicate),
					"Field":    Equal("providerConfig.mirrors[0].hosts[0].capabilities[2]"),
					"BadValue": Equal("pull"),
				})),
			))
		})

		It("should deny duplicate mirror upstreams", func() {
			mirrorConfig.Mirrors = append(mirrorConfig.Mirrors, *mirrorConfig.Mirrors[0].DeepCopy())

			Expect(ValidateMirrorConfig(mirrorConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("providerConfig.mirrors[1].upstream"),
				})),
			))
		})
	})

	Describe("#ValidateMirrorHostCABundleSecret", func() {
		const caBundle = "-----BEGIN CERTIFICATE-----\nMIICRzCCAfGgAwIBAgIJALMb7ecMIk3MMA0GCSqGSIb3DQEBCwUAMH4xCzAJBgNV\nBAYTAkdCMQ8wDQYDVQQIDAZMb25kb24xDzANBgNVBAcMBkxvbmRvbjEYMBYGA1UE\nCgwPR2xvYmFsIFNlY3VyaXR5MRYwFAYDVQQLDA1JVCBEZXBhcnRtZW50MRswGQYD\nVQQDDBJ0ZXN0LWNlcnRpZmljYXRlLTAwIBcNMTcwNDI2MjMyNjUyWhgPMjExNzA0\nMDIyMzI2NTJaMH4xCzAJBgNVBAYTAkdCMQ8wDQYDVQQIDAZMb25kb24xDzANBgNV\nBAcMBkxvbmRvbjEYMBYGA1UECgwPR2xvYmFsIFNlY3VyaXR5MRYwFAYDVQQLDA1J\nVCBEZXBhcnRtZW50MRswGQYDVQQDDBJ0ZXN0LWNlcnRpZmljYXRlLTAwXDANBgkq\nhkiG9w0BAQEFAANLADBIAkEAtBMa7NWpv3BVlKTCPGO/LEsguKqWHBtKzweMY2CV\ntAL1rQm913huhxF9w+ai76KQ3MHK5IVnLJjYYA5MzP2H5QIDAQABo1AwTjAdBgNV\nHQ4EFgQU22iy8aWkNSxv0nBxFxerfsvnZVMwHwYDVR0jBBgwFoAU22iy8aWkNSxv\n0nBxFxerfsvnZVMwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAANBAEOefGbV\nNcHxklaW06w6OBYJPwpIhCVozC1qdxGX1dg8VkEKzjOzjgqVD30m59OFmSlBmHsl\nnkVA6wyOSDYBf3o=\n-----END CERTIFICATE-----"

		var secret *corev1.Secret

		BeforeEach(func() {
			fldPath = fldPath.Child("mirrors").Index(0).Child("hosts").Index(0).Child("caBundleSecretReferenceName")
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "bar",
				},
				Immutable: ptr.To(true),
				Data: map[string][]byte{
					"bundle.crt": []byte(caBundle),
				},
			}
		})

		It("should allow valid CA bundle secret", func() {
			Expect(ValidateMirrorHostCABundleSecret(secret, fldPath, "foo-secret-ref")).To(BeEmpty())
		})

		DescribeTable("should deny secrets which are not immutable",
			func(isImmutable *bool) {
				secret.Immutable = isImmutable

				Expect(ValidateMirrorHostCABundleSecret(secret, fldPath, "foo-secret-ref")).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":     Equal(field.ErrorTypeInvalid),
						"Field":    Equal("providerConfig.mirrors[0].hosts[0].caBundleSecretReferenceName"),
						"BadValue": Equal("foo-secret-ref"),
						"Detail":   ContainSubstring(`the referenced CA bundle secret "foo/bar" should be immutable`),
					})),
				))
			},
			Entry("when immutable field is nil", nil),
			Entry("when immutable field is false", ptr.To(false)),
		)

		It("should deny secret without 'bundle.crt' data entry", func() {
			delete(secret.Data, "bundle.crt")

			Expect(ValidateMirrorHostCABundleSecret(secret, fldPath, "foo-secret-ref")).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.mirrors[0].hosts[0].caBundleSecretReferenceName"),
					"BadValue": Equal("foo-secret-ref"),
					"Detail":   Equal(`missing "bundle.crt" data entry in the referenced CA bundle secret "foo/bar"`),
				})),
			))
		})

		It("should only have a single data entry", func() {
			secret.Data["foo"] = []byte("bar")

			Expect(ValidateMirrorHostCABundleSecret(secret, fldPath, "foo-secret-ref")).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.mirrors[0].hosts[0].caBundleSecretReferenceName"),
					"BadValue": Equal("foo-secret-ref"),
					"Detail":   ContainSubstring(`the referenced CA bundle secret "foo/bar" should only have a single data entry with key "bundle.crt"`),
				})),
			))
		})

		It("should deny secret with invalid CA bundle", func() {
			secret.Data["bundle.crt"] = []byte("bar")

			Expect(ValidateMirrorHostCABundleSecret(secret, fldPath, "foo-secret-ref")).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.mirrors[0].hosts[0].caBundleSecretReferenceName"),
					"BadValue": Equal("foo-secret-ref"),
					"Detail":   ContainSubstring("the CA bundle is not a valid PEM-encoded certificate"),
				})),
			))
		})
	})
})
