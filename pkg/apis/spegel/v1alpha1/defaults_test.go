// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/spegel/v1alpha1"
)

var _ = Describe("Defaults", func() {

	Describe("SpegelConfig defaulting", func() {
		It("should default correctly", func() {
			obj := &v1alpha1.SpegelConfig{}

			v1alpha1.SetObjectDefaults_SpegelConfig(obj)

			expected := &v1alpha1.SpegelConfig{
				RegistryPort: new(int32(15500)),
				RouterPort:   new(int32(15501)),
				MetricsPort:  new(int32(19090)),
				ResolveTags:  new(bool(true)),
			}
			Expect(obj).To(Equal(expected))
		})

		It("should not overwrite already set values", func() {
			obj := &v1alpha1.SpegelConfig{
				RegistryPort: new(int32(5500)),
				RouterPort:   new(int32(5501)),
				MetricsPort:  new(int32(9090)),
				ResolveTags:  new(bool(false)),
			}
			expected := obj.DeepCopy()

			v1alpha1.SetObjectDefaults_SpegelConfig(obj)

			Expect(obj).To(Equal(expected))
		})
	})
})
