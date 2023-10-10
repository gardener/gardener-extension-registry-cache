// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation/field"

	api "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	. "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/validation"
)

var _ = Describe("Validation", func() {
	var (
		fldPath = field.NewPath("providerConfig")

		registryConfig *api.RegistryConfig
	)

	BeforeEach(func() {
		size := resource.MustParse("5Gi")
		registryConfig = &api.RegistryConfig{
			Caches: []api.RegistryCache{{
				Upstream: "docker.io",
				Size:     &size,
			}},
		}
	})

	Describe("#ValidateRegistryConfig", func() {
		It("should allow valid configuration", func() {
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

		It("should require upstream", func() {
			registryConfig.Caches[0].Upstream = ""

			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("providerConfig.caches[0].upstream"),
					"Detail": ContainSubstring("upstream must be provided"),
				})),
			))
		})

		It("should deny upstream with scheme", func() {
			cache := api.RegistryCache{
				Upstream: "http://docker.io",
			}
			registryConfig.Caches = append(registryConfig.Caches, cache)

			registryConfig.Caches[0].Upstream = "https://docker.io"

			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("providerConfig.caches[0].upstream"),
					"Detail": ContainSubstring("upstream must not include a scheme"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("providerConfig.caches[1].upstream"),
					"Detail": ContainSubstring("upstream must not include a scheme"),
				})),
			))
		})

		It("should deny non-positive cache size", func() {
			negativeSize := resource.MustParse("-1Gi")
			cache := api.RegistryCache{
				Upstream: "quay.io",
				Size:     &negativeSize,
			}
			registryConfig.Caches = append(registryConfig.Caches, cache)

			zeroSize := resource.MustParse("0")
			registryConfig.Caches[0].Size = &zeroSize

			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("providerConfig.caches[0].size"),
					"Detail": ContainSubstring("must be greater than 0"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("providerConfig.caches[1].size"),
					"Detail": ContainSubstring("must be greater than 0"),
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
	})

	Describe("#ValidateRegistryConfigUpdate", func() {
		var oldRegistryConfig *api.RegistryConfig

		BeforeEach(func() {
			oldRegistryConfig = registryConfig.DeepCopy()
		})

		It("should allow valid configuration update", func() {
			size := resource.MustParse("5Gi")
			newCache := api.RegistryCache{
				Upstream: "docker.io",
				Size:     &size,
				GarbageCollection: &api.GarbageCollection{
					Enabled: true,
				},
			}
			registryConfig.Caches = append(registryConfig.Caches, newCache)

			Expect(ValidateRegistryConfigUpdate(oldRegistryConfig, registryConfig, fldPath)).To(BeEmpty())
		})

		It("should deny cache size update", func() {
			newSize := resource.MustParse("16Gi")
			registryConfig.Caches[0].Size = &newSize

			Expect(ValidateRegistryConfigUpdate(oldRegistryConfig, registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("providerConfig.caches[0].size"),
					"Detail": Equal("field is immutable"),
				})),
			))
		})
	})
})
