// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha2_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha2"
)

var _ = Describe("Conversions", func() {

	var (
		scheme *runtime.Scheme
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(v1alpha2.AddToScheme(scheme)).To(Succeed())
	})

	Describe("#Convert_v1alpha2_GarbageCollection_To_registry_GarbageCollection", func() {
		It("should convert successfully when enabled=true", func() {
			in := &v1alpha2.GarbageCollection{
				Enabled: true,
			}
			out := &registry.GarbageCollection{}

			Expect(scheme.Convert(in, out, nil)).To(Succeed())

			expected := &registry.GarbageCollection{
				TTL: metav1.Duration{Duration: 7 * 24 * time.Hour},
			}
			Expect(out).To(Equal(expected))
		})

		It("should convert successfully when enabled=false", func() {
			in := &v1alpha2.GarbageCollection{
				Enabled: false,
			}
			out := &registry.GarbageCollection{}

			Expect(scheme.Convert(in, out, nil)).To(Succeed())

			expected := &registry.GarbageCollection{
				TTL: metav1.Duration{Duration: 0},
			}
			Expect(out).To(Equal(expected))
		})
	})

	Describe("#Convert_registry_GarbageCollection_To_v1alpha2_GarbageCollection", func() {
		It("should convert successfully when ttl > 0", func() {
			in := &registry.GarbageCollection{
				TTL: metav1.Duration{Duration: 7 * 24 * time.Hour},
			}
			out := &v1alpha2.GarbageCollection{}

			Expect(scheme.Convert(in, out, nil)).To(Succeed())

			expected := &v1alpha2.GarbageCollection{
				Enabled: true,
			}
			Expect(out).To(Equal(expected))
		})

		It("should convert successfully when ttl = 0", func() {
			in := &registry.GarbageCollection{
				TTL: metav1.Duration{Duration: 0},
			}
			out := &v1alpha2.GarbageCollection{}

			Expect(scheme.Convert(in, out, nil)).To(Succeed())

			expected := &v1alpha2.GarbageCollection{
				Enabled: false,
			}
			Expect(out).To(Equal(expected))
		})
	})
})
