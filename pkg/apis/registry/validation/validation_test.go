// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	api "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	. "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/validation"
)

var _ = Describe("Validation", func() {
	var (
		fldPath        *field.Path
		registryConfig *api.RegistryConfig
	)

	BeforeEach(func() {
		fldPath = field.NewPath("providerConfig")
		size := resource.MustParse("5Gi")
		registryConfig = &api.RegistryConfig{
			Caches: []api.RegistryCache{
				{
					Upstream: "docker.io",
					Volume: &api.Volume{
						Size: &size,
					},
				},
			},
		}
	})

	Describe("#ValidateRegistryConfig", func() {
		It("should allow valid configuration", func() {
			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(BeEmpty())
		})

		It("should allow valid remoteURLs", func() {
			registryConfig.Caches[0].RemoteURL = ptr.To("https://registry-1.docker.io")
			registryConfig.Caches = append(registryConfig.Caches,
				api.RegistryCache{
					Upstream:  "my-registry.io",
					RemoteURL: ptr.To("https://my-registry.io"),
				},
				api.RegistryCache{
					Upstream:  "my-registry.io:5000",
					RemoteURL: ptr.To("http://my-registry.io:5000"),
					Proxy: &api.Proxy{
						HTTPProxy:  ptr.To("http://127.0.0.1"),
						HTTPSProxy: ptr.To("https://127.0.0.1:1234"),
					},
				},
				api.RegistryCache{
					Upstream:  "quay.io",
					RemoteURL: ptr.To("https://mirror-host.io:8443"),
				},
			)
			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(BeEmpty())
		})

		It("should deny configuration without a cache", func() {
			registryConfig = &api.RegistryConfig{Caches: nil}
			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("providerConfig.caches"),
					"Detail": ContainSubstring("at least one cache must be provided"),
				})),
			))

			registryConfig = &api.RegistryConfig{Caches: []api.RegistryCache{}}
			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("providerConfig.caches"),
					"Detail": ContainSubstring("at least one cache must be provided"),
				})),
			))
		})

		It("should deny invalid upstreams", func() {
			registryConfig.Caches[0].Upstream = ""

			registryConfig.Caches = append(registryConfig.Caches,
				api.RegistryCache{
					Upstream: "docker.io.",
				},
				api.RegistryCache{
					Upstream: ".docker.io",
				},
				api.RegistryCache{
					Upstream: "https://docker.io",
				},
				api.RegistryCache{
					Upstream: "docker.io:0443",
				},
			)

			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[0].upstream"),
					"BadValue": Equal(""),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[1].upstream"),
					"BadValue": Equal("docker.io."),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[2].upstream"),
					"BadValue": Equal(".docker.io"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[3].upstream"),
					"BadValue": Equal("https://docker.io"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[4].upstream"),
					"BadValue": Equal("docker.io:0443"),
				})),
			))
		})

		It("should deny non-positive cache size", func() {
			negativeSize := resource.MustParse("-1Gi")
			cache := api.RegistryCache{
				Upstream: "myproj-releases.common.repositories.cloud.com",
				Volume: &api.Volume{
					Size: &negativeSize,
				},
			}
			registryConfig.Caches = append(registryConfig.Caches, cache)

			zeroSize := resource.MustParse("0")
			registryConfig.Caches[0].Volume.Size = &zeroSize

			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("providerConfig.caches[0].volume.size"),
					"Detail": ContainSubstring("must be greater than 0"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("providerConfig.caches[1].volume.size"),
					"Detail": ContainSubstring("must be greater than 0"),
				})),
			))
		})

		It("should deny invalid storage class names", func() {
			cache := api.RegistryCache{
				Upstream: "myproj-releases.common.repositories.cloud.com",
				Volume: &api.Volume{
					StorageClassName: ptr.To("invalid/name"),
				},
			}
			registryConfig.Caches = append(registryConfig.Caches, cache)

			tooLongStorageClassName := strings.Repeat("n", 254)
			registryConfig.Caches[0].Volume.StorageClassName = ptr.To(tooLongStorageClassName)

			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[0].volume.storageClassName"),
					"BadValue": Equal(tooLongStorageClassName),
					"Detail":   ContainSubstring("must be no more than 253 characters"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[1].volume.storageClassName"),
					"BadValue": Equal("invalid/name"),
					"Detail":   ContainSubstring("must consist of lower case alphanumeric characters, '-' or '.'"),
				})),
			))
		})

		It("should deny negative garbage collection ttl duration", func() {
			registryConfig.Caches[0].GarbageCollection = &api.GarbageCollection{
				TTL: metav1.Duration{Duration: -1 * time.Hour},
			}

			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("providerConfig.caches[0].garbageCollection.ttl"),
					"Detail": ContainSubstring("ttl must be a non-negative duration"),
				})),
			))
		})

		It("should deny duplicate cache upstreams", func() {
			registryConfig.Caches = append(registryConfig.Caches, *registryConfig.Caches[0].DeepCopy())

			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("providerConfig.caches[1].upstream"),
				})),
			))
		})

		It("should deny invalid remoteURLs", func() {
			registryConfig.Caches[0].RemoteURL = ptr.To("ftp://docker.io")
			registryConfig.Caches = append(registryConfig.Caches,
				api.RegistryCache{
					Upstream:  "my-registry.io:5000",
					RemoteURL: ptr.To("http://my-registry.io:5000/repository"),
				},
				api.RegistryCache{
					Upstream:  "my-registry.io:8443",
					RemoteURL: ptr.To("https://my-registry.io:8443/repository"),
				},
				api.RegistryCache{
					Upstream:  "quay.io",
					RemoteURL: ptr.To("mirror-host.io:8443"),
				},
			)
			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[0].remoteURL"),
					"BadValue": Equal("ftp://docker.io"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[1].remoteURL"),
					"BadValue": Equal("http://my-registry.io:5000/repository"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[2].remoteURL"),
					"BadValue": Equal("https://my-registry.io:8443/repository"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[3].remoteURL"),
					"BadValue": Equal("mirror-host.io:8443"),
				})),
			))
		})

		It("should deny invalid proxy config", func() {
			registryConfig.Caches[0].Proxy = &api.Proxy{
				HTTPProxy:  ptr.To("10.10.10.10"),
				HTTPSProxy: nil,
			}
			registryConfig.Caches = append(registryConfig.Caches,
				api.RegistryCache{
					Upstream: "my-registry.io",
					Proxy: &api.Proxy{
						HTTPProxy:  nil,
						HTTPSProxy: ptr.To("http://foo!bar"),
					},
				},
				api.RegistryCache{
					Upstream: "my-registry2.io",
					Proxy: &api.Proxy{
						HTTPProxy:  nil,
						HTTPSProxy: nil,
					},
				},
			)
			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[0].proxy.httpProxy"),
					"BadValue": Equal("10.10.10.10"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[1].proxy.httpsProxy"),
					"BadValue": Equal("http://foo!bar"),
				})),
			))
		})
	})

	Describe("#ValidateRegistryConfigUpdate", func() {
		var oldRegistryConfig *api.RegistryConfig

		BeforeEach(func() {
			oldRegistryConfig = registryConfig.DeepCopy()
		})

		It("should allow valid configuration update", func() {
			registryConfig.Caches[0].GarbageCollection = &api.GarbageCollection{
				TTL: metav1.Duration{Duration: 14 * 24 * time.Hour},
			}
			size := resource.MustParse("5Gi")
			newCache := api.RegistryCache{
				Upstream: "quay.io",
				Volume: &api.Volume{
					Size: &size,
				},
			}
			registryConfig.Caches = append(registryConfig.Caches, newCache)

			Expect(ValidateRegistryConfigUpdate(oldRegistryConfig, registryConfig, fldPath)).To(BeEmpty())
		})

		It("should deny cache volume size update", func() {
			newSize := resource.MustParse("16Gi")
			registryConfig.Caches[0].Volume.Size = &newSize

			Expect(ValidateRegistryConfigUpdate(oldRegistryConfig, registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[0].volume.size"),
					"BadValue": Equal("16Gi"),
					"Detail":   Equal("field is immutable"),
				})),
			))
		})

		It("should deny cache volume storageClassName update", func() {
			registryConfig.Caches[0].Volume.StorageClassName = ptr.To("foo")

			Expect(ValidateRegistryConfigUpdate(oldRegistryConfig, registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[0].volume.storageClassName"),
					"BadValue": Equal(ptr.To("foo")),
					"Detail":   Equal("field is immutable"),
				})),
			))
		})

		It("should deny garbage collection enablement (ttl > 0) once it is disabled (ttl = 0)", func() {
			oldRegistryConfig.Caches[0].GarbageCollection = &api.GarbageCollection{
				TTL: metav1.Duration{Duration: 0},
			}
			registryConfig.Caches[0].GarbageCollection = &api.GarbageCollection{
				TTL: metav1.Duration{Duration: 7 * 24 * time.Hour},
			}

			Expect(ValidateRegistryConfigUpdate(oldRegistryConfig, registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("providerConfig.caches[0].garbageCollection.ttl"),
					"Detail": Equal("garbage collection cannot be enabled (ttl > 0) once it is disabled (ttl = 0)"),
				})),
			))
		})
	})

	Describe("#ValidateUpstreamRegistrySecret", func() {
		var secret *corev1.Secret

		BeforeEach(func() {
			fldPath = fldPath.Child("caches").Index(0).Child("secretReferenceName")
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "bar",
				},
				Data: map[string][]byte{
					"username": []byte("john"),
					"password": []byte("swordfish"),
				},
				Immutable: ptr.To(true),
			}
		})

		It("should allow valid upstream registry secret", func() {
			Expect(ValidateUpstreamRegistrySecret(secret, fldPath, "foo-secret-ref")).To(BeEmpty())
		})

		DescribeTable("should deny non immutable secrets",
			func(isImmutable *bool) {
				secret.Immutable = isImmutable

				Expect(ValidateUpstreamRegistrySecret(secret, fldPath, "foo-secret-ref")).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("providerConfig.caches[0].secretReferenceName"),
						"Detail": ContainSubstring(`referenced secret "foo/bar" should be immutable`),
					})),
				))
			},
			Entry("when immutable field is nil", nil),
			Entry("when immutable field is false", ptr.To(false)),
		)

		DescribeTable("should have only two data entries",
			func(data map[string][]byte) {
				secret.Data = data

				Expect(ValidateUpstreamRegistrySecret(secret, fldPath, "foo-secret-ref")).To(ContainElements(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("providerConfig.caches[0].secretReferenceName"),
						"Detail": ContainSubstring(`referenced secret "foo/bar" should have only two data entries`),
					})),
				))
			},
			Entry("when secret data is empty", map[string][]byte{}),
			Entry("when secret data has more entries", map[string][]byte{
				"username": []byte("john"),
				"password": []byte("swordfish"),
				"foo":      []byte("foo"),
			}),
		)

		It("should deny secrets without 'username' data entry", func() {
			delete(secret.Data, "username")

			Expect(ValidateUpstreamRegistrySecret(secret, fldPath, "foo-secret-ref")).To(ContainElements(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("providerConfig.caches[0].secretReferenceName"),
					"Detail": Equal(`missing "username" data entry in referenced secret "foo/bar"`),
				})),
			))
		})

		It("should deny secrets with empty 'username' data entry", func() {
			secret.Data["username"] = []byte("")

			Expect(ValidateUpstreamRegistrySecret(secret, fldPath, "foo-secret-ref")).To(ContainElements(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("providerConfig.caches[0].secretReferenceName"),
					"Detail": Equal(`data entry "username" in referenced secret "foo/bar" is empty`),
				})),
			))
		})

		It("should deny secrets when 'username' data entry contains whitespaces", func() {
			secret.Data["username"] = []byte("us	er")

			Expect(ValidateUpstreamRegistrySecret(secret, fldPath, "foo-secret-ref")).To(ContainElements(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("providerConfig.caches[0].secretReferenceName"),
					"Detail": Equal(`data entry "username" in referenced secret "foo/bar" contains whitespace`),
				})),
			))
		})

		It("should deny secrets without 'password' data entry", func() {
			delete(secret.Data, "password")

			Expect(ValidateUpstreamRegistrySecret(secret, fldPath, "foo-secret-ref")).To(ContainElements(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("providerConfig.caches[0].secretReferenceName"),
					"Detail": Equal(`missing "password" data entry in referenced secret "foo/bar"`),
				})),
			))
		})

		It("should deny secrets with empty 'password' data entry", func() {
			secret.Data["password"] = []byte(" 	")

			Expect(ValidateUpstreamRegistrySecret(secret, fldPath, "foo-secret-ref")).To(ContainElements(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.caches[0].secretReferenceName"),
					"BadValue": Equal("foo-secret-ref"),
					"Detail":   Equal(`data entry "password" in referenced secret "foo/bar" is empty`),
				})),
			))
		})

		When("user is '_json_key' and password is ServiceAccount json", func() {
			BeforeEach(func() {
				secret.Data["username"] = []byte("_json_key")
				secret.Data["password"] = []byte(`{
    "type": "service_account",
    "project_id": "foo",
    "private_key_id": "a1b2c3d4",
    "private_key": "-----BEGIN PRIVATE KEY-----\n<private-key-content>\n-----END PRIVATE KEY-----\n",
    "client_email": "foo@bar.com",
    "client_id": "1234",
    "auth_uri": "https://accounts.google.com/o/oauth2/auth",
    "token_uri": "https://oauth2.googleapis.com/token",
    "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
    "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/foo%40bar.com",
    "universe_domain": "googleapis.com"
}`)
			})

			It("should allow secrets with valid ServiceAccount json", func() {
				Expect(ValidateUpstreamRegistrySecret(secret, fldPath, "foo-secret-ref")).To(BeEmpty())
			})

			It("should deny a secret with invalid ServiceAccount json", func() {
				secret.Data["password"] = []byte(`{
    "type": "service_account",
    "project_id": "foo",
    "private_key_id": "a1b2c3d4",
    "private_key": "-----BEGIN PRIVATE KEY-----
<private-key-content>
-----END PRIVATE KEY-----
",
    "client_email": "foo@bar.com",
    "client_id": "1234",
    "auth_uri": "https://accounts.google.com/o/oauth2/auth",
    "token_uri": "https://oauth2.googleapis.com/token",
    "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
    "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/foo%40bar.com",
    "universe_domain": "googleapis.com"
}`)
				Expect(ValidateUpstreamRegistrySecret(secret, fldPath, "foo-secret-ref")).To(ContainElements(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("providerConfig.caches[0].secretReferenceName"),
						"Detail": Equal(`failed to unmarshal ServiceAccount json from password data entry in referenced secret "foo/bar": invalid character '\n' in string literal`),
					})),
				))
			})

			It("should deny a secret with forbidden ServiceAccount fields", func() {
				secret.Data["password"] = []byte(`{
    "auths": {
        "europe-docker.pkg.dev": {
            "auth": "<auth-content>"
        }
    },
    "baz": "qux"
}`)
				Expect(ValidateUpstreamRegistrySecret(secret, fldPath, "foo-secret-ref")).To(ContainElements(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("providerConfig.caches[0].secretReferenceName"),
						"Detail": Equal(`forbidden ServiceAccount field "auths" present in password data entry in referenced secret "foo/bar"`),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("providerConfig.caches[0].secretReferenceName"),
						"Detail": Equal(`forbidden ServiceAccount field "baz" present in password data entry in referenced secret "foo/bar"`),
					})),
				))
			})
		})
	})

	Describe("#ValidateUpstream", func() {
		BeforeEach(func() {
			fldPath = fldPath.Child("caches").Index(0).Child("upstream")
		})

		DescribeTable("should allow valid upstreams",
			func(upstream string) {
				Expect(ValidateUpstream(fldPath, upstream)).To(BeEmpty())
			},
			Entry("when host is valid", "example.com"),
			Entry("when port is set", "example.com:5000"),
			Entry("when port is 1", "example.com:1"),
			Entry("when port is 65535", "example.com:65535"),
		)

		DescribeTable("should deny invalid upstreams",
			func(upstream string) {
				Expect(ValidateUpstream(fldPath, upstream)).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":     Equal(field.ErrorTypeInvalid),
						"BadValue": Equal(upstream),
					})),
				))
			},
			Entry("when starts with https scheme", "https://example.com"),
			Entry("when starts with http scheme", "http://example.com:5000"),
			Entry("when scheme is not supported", "ftp://example.com:5000"),
			Entry("when port is invalid", "example.com:0123"),
			Entry("when port is 0", "example.com:0"),
			Entry("when port is out of range", "example.com:65536"),
			Entry("when host is very long", strings.Repeat("n", 250)+".com"),
			Entry("when query param is set", "example.com?foo=bar"),
			Entry("when path is set", "example.com/foo/bar"),
		)
	})

	Describe("#ValidateURL", func() {
		BeforeEach(func() {
			fldPath = fldPath.Child("caches").Index(0).Child("remoteURL")
		})

		DescribeTable("should allow valid urls",
			func(upstream string) {
				Expect(ValidateURL(fldPath, upstream)).To(BeEmpty())
			},
			Entry("when url consists of valid scheme and host", "https://example.com"),
			Entry("when url consists of valid scheme, host and port", "http://example.com:5000"),
		)

		DescribeTable("should deny invalid urls",
			func(upstream string) {
				Expect(ValidateURL(fldPath, upstream)).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":     Equal(field.ErrorTypeInvalid),
						"BadValue": Equal(upstream),
					})),
				))
			},
			Entry("when scheme is missing", "example.com"),
			Entry("when scheme is not supported", "ftp://example.com"),
			Entry("when port is invalid", "https://example.com:80443"),
			Entry("when path is set", "https://example.com/myrepository"),
			Entry("when query param is set", "https://example.com?foo=bar"),
			Entry("when user is set", "https://foo:bar@example.com/myrepository"),
		)
	})
})
