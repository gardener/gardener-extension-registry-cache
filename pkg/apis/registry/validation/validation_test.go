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
	"k8s.io/utils/pointer"

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
				Upstream:                 "docker.io",
				Size:                     &size,
				GarbageCollectionEnabled: pointer.Bool(true),
			}},
		}
	})

	Describe("#ValidateRegistryConfig", func() {
		It("should allow valid configuration", func() {
			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(BeEmpty())
		})

		It("should require upstream", func() {
			registryConfig.Caches[0].Upstream = ""

			path := fldPath.Child("caches").Index(0).Child("upstream").String()
			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal(path),
					"Detail": ContainSubstring("upstream must be provided"),
				})),
			))
		})

		It("should deny upstream with scheme", func() {
			registryConfig.Caches = append(registryConfig.Caches, *registryConfig.Caches[0].DeepCopy())
			registryConfig.Caches[0].Upstream = "https://docker.io"
			registryConfig.Caches[1].Upstream = "http://docker.io"

			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal(fldPath.Child("caches").Index(0).Child("upstream").String()),
					"Detail": ContainSubstring("upstream must not include a scheme"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal(fldPath.Child("caches").Index(1).Child("upstream").String()),
					"Detail": ContainSubstring("upstream must not include a scheme"),
				})),
			))
		})

		It("should deny non-positive cache size", func() {
			registryConfig.Caches = append(registryConfig.Caches, *registryConfig.Caches[0].DeepCopy())
			zeroSize := resource.MustParse("0")
			negativeSize := resource.MustParse("-1Gi")
			registryConfig.Caches[0].Size = &zeroSize
			registryConfig.Caches[1].Size = &negativeSize

			Expect(ValidateRegistryConfig(registryConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal(fldPath.Child("caches").Index(0).Child("size").String()),
					"Detail": ContainSubstring("size must be a quantity greater than zero"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal(fldPath.Child("caches").Index(1).Child("size").String()),
					"Detail": ContainSubstring("size must be a quantity greater than zero"),
				})),
			))
		})
	})
})
