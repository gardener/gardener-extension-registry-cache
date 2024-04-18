// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	api "github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror"
	. "github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror/validation"
)

var _ = Describe("Validation", func() {
	var (
		fldPath      *field.Path
		mirrorConfig *api.MirrorConfig
	)

	BeforeEach(func() {
		fldPath = field.NewPath("providerConfig")
		mirrorConfig = &api.MirrorConfig{
			Mirrors: []api.MirrorConfiguration{
				{
					Upstream: "docker.io",
					Hosts: []api.MirrorHost{
						{
							Host:         "https://mirror.gcr.io",
							Capabilities: []api.MirrorHostCapability{api.MirrorHostCapabilityPull, api.MirrorHostCapabilityResolve},
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

		It("should deny configuration without a mirror", func() {
			mirrorConfig = &api.MirrorConfig{Mirrors: nil}
			Expect(ValidateMirrorConfig(mirrorConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("providerConfig.mirrors"),
					"Detail": ContainSubstring("at least one mirror must be provided"),
				})),
			))

			mirrorConfig = &api.MirrorConfig{Mirrors: []api.MirrorConfiguration{}}
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
				api.MirrorConfiguration{
					Upstream: "docker.io.",
					Hosts:    []api.MirrorHost{{Host: "https://mirror.gcr.io"}},
				},
				api.MirrorConfiguration{
					Upstream: ".docker.io",
					Hosts:    []api.MirrorHost{{Host: "https://mirror.gcr.io"}},
				},
				api.MirrorConfiguration{
					Upstream: "https://docker.io",
					Hosts:    []api.MirrorHost{{Host: "https://mirror.gcr.io"}},
				},
				api.MirrorConfiguration{
					Upstream: "docker.io:0443",
					Hosts:    []api.MirrorHost{{Host: "https://mirror.gcr.io"}},
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
			mirrorConfig = &api.MirrorConfig{
				Mirrors: []api.MirrorConfiguration{
					{
						Upstream: "docker.io",
						Hosts: []api.MirrorHost{
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
			mirrorConfig = &api.MirrorConfig{
				Mirrors: []api.MirrorConfiguration{
					{
						Upstream: "docker.io",
						Hosts: []api.MirrorHost{
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
			mirrorConfig = &api.MirrorConfig{
				Mirrors: []api.MirrorConfiguration{
					{
						Upstream: "docker.io",
						Hosts: []api.MirrorHost{
							{
								Host:         "https://mirror.gcr.io",
								Capabilities: []api.MirrorHostCapability{"foo"},
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
			mirrorConfig = &api.MirrorConfig{
				Mirrors: []api.MirrorConfiguration{
					{
						Upstream: "docker.io",
						Hosts: []api.MirrorHost{
							{
								Host:         "https://mirror.gcr.io",
								Capabilities: []api.MirrorHostCapability{"pull", "resolve", "pull"},
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
})
