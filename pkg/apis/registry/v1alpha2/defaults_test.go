// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha2"
)

var _ = Describe("Defaults", func() {

	var (
		defaultSize = resource.MustParse("10Gi")
	)

	Describe("RegistryCache defaulting", func() {
		It("should default correctly", func() {
			obj := &v1alpha2.RegistryConfig{
				Caches: []v1alpha2.RegistryCache{
					{},
				},
			}

			v1alpha2.SetObjectDefaults_RegistryConfig(obj)

			expected := &v1alpha2.RegistryConfig{
				Caches: []v1alpha2.RegistryCache{
					{
						Volume: &v1alpha2.Volume{
							Size: &defaultSize,
						},
						GarbageCollection: &v1alpha2.GarbageCollection{
							Enabled: true,
						},
					},
				},
			}
			Expect(obj).To(Equal(expected))
		})

		It("should not overwrite already set values", func() {
			customSize := resource.MustParse("20Gi")
			obj := &v1alpha2.RegistryConfig{
				Caches: []v1alpha2.RegistryCache{
					{
						Volume: &v1alpha2.Volume{
							Size: &customSize,
						},
						GarbageCollection: &v1alpha2.GarbageCollection{
							Enabled: false,
						},
					},
				},
			}
			expected := obj.DeepCopy()

			v1alpha2.SetObjectDefaults_RegistryConfig(obj)

			Expect(obj).To(Equal(expected))
		})
	})
})
