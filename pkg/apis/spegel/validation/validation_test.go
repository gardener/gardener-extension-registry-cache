// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	spegelapi "github.com/gardener/gardener-extension-registry-cache/pkg/apis/spegel"
	. "github.com/gardener/gardener-extension-registry-cache/pkg/apis/spegel/validation"
)

var _ = Describe("Validation", func() {
	var (
		fldPath      *field.Path
		spegelConfig *spegelapi.SpegelConfig
	)

	BeforeEach(func() {
		fldPath = field.NewPath("providerConfig")
		spegelConfig = &spegelapi.SpegelConfig{
			RegistryPort: new(int32(15500)),
			RouterPort:   new(int32(15501)),
			MetricsPort:  new(int32(19090)),
			ResolveTags:  new(bool(true)),
		}
	})

	Describe("#ValidateSpegelConfig", func() {
		It("should allow valid configuration", func() {
			Expect(ValidateSpegelConfig(spegelConfig, fldPath)).To(BeEmpty())
		})

		It("should require the registry port", func() {
			spegelConfig.RegistryPort = nil

			Expect(ValidateSpegelConfig(spegelConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("providerConfig.registryPort"),
				})),
			))
		})

		It("should require the router port", func() {
			spegelConfig.RouterPort = nil

			Expect(ValidateSpegelConfig(spegelConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("providerConfig.routerPort"),
				})),
			))
		})

		It("should require the metrics port", func() {
			spegelConfig.MetricsPort = nil

			Expect(ValidateSpegelConfig(spegelConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("providerConfig.metricsPort"),
				})),
			))
		})

		It("should deny a port that is too low", func() {
			spegelConfig.RegistryPort = new(int32(0))

			Expect(ValidateSpegelConfig(spegelConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.registryPort"),
					"BadValue": Equal(int32(0)),
				})),
			))
		})

		It("should deny a port that is too high", func() {
			spegelConfig.MetricsPort = new(int32(70000))

			Expect(ValidateSpegelConfig(spegelConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.metricsPort"),
					"BadValue": Equal(int32(70000)),
				})),
			))
		})

		It("should deny colliding registry and router ports", func() {
			spegelConfig.RouterPort = new(int32(15500))

			Expect(ValidateSpegelConfig(spegelConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("providerConfig.routerPort"),
					"BadValue": Equal(int32(15500)),
					"Detail":   ContainSubstring("must not be equal to the other Spegel ports"),
				})),
			))
		})

		It("should deny all three ports being equal", func() {
			spegelConfig.RegistryPort = new(int32(15500))
			spegelConfig.RouterPort = new(int32(15500))
			spegelConfig.MetricsPort = new(int32(15500))

			Expect(ValidateSpegelConfig(spegelConfig, fldPath)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("providerConfig.routerPort"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("providerConfig.metricsPort"),
				})),
			))
		})
	})
})
