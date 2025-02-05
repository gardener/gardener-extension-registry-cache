// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha3_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
)

var _ = Describe("Defaults", func() {

	var (
		defaultSize = resource.MustParse("10Gi")
	)

	Describe("RegistryCache defaulting", func() {
		It("should default correctly", func() {
			obj := &v1alpha3.RegistryConfig{
				Caches: []v1alpha3.RegistryCache{
					{},
				},
			}

			v1alpha3.SetObjectDefaults_RegistryConfig(obj)

			expected := &v1alpha3.RegistryConfig{
				Caches: []v1alpha3.RegistryCache{
					{
						Volume: &v1alpha3.Volume{
							Size: &defaultSize,
						},
						GarbageCollection: &v1alpha3.GarbageCollection{
							TTL: metav1.Duration{Duration: 7 * 24 * time.Hour},
						},
						HTTP: &v1alpha3.HTTP{
							TLS: true,
						},
					},
				},
			}
			Expect(obj).To(Equal(expected))
		})

		It("should not overwrite already set values", func() {
			customSize := resource.MustParse("20Gi")
			obj := &v1alpha3.RegistryConfig{
				Caches: []v1alpha3.RegistryCache{
					{
						Volume: &v1alpha3.Volume{
							Size: &customSize,
						},
						GarbageCollection: &v1alpha3.GarbageCollection{
							TTL: metav1.Duration{Duration: 0},
						},
						HTTP: &v1alpha3.HTTP{
							TLS: false,
						},
					},
				},
			}
			expected := obj.DeepCopy()

			v1alpha3.SetObjectDefaults_RegistryConfig(obj)

			Expect(obj).To(Equal(expected))
		})
	})
})
