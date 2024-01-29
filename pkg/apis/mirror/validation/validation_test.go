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
						{Host: "https://mirror.gcr.io"},
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
					Upstream: "docker.io:443",
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
					"BadValue": Equal("docker.io:443"),
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
					"Detail":   Equal("mirror must include scheme 'http://' or 'https://'"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.mirrors[0].hosts[1].host"),
					"BadValue": Equal("docker-mirror.internal"),
					"Detail":   Equal("mirror must include scheme 'http://' or 'https://'"),
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
