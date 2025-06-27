// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/config"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/config/validation"
)

var _ = Describe("Validation", func() {
	DescribeTable("#ValidateConfiguration",
		func(config config.Configuration, match gomegatypes.GomegaMatcher) {
			err := validation.ValidateConfiguration(&config)
			Expect(err).To(match)
		},
		Entry("config", config.Configuration{}, BeEmpty()),
	)
})
