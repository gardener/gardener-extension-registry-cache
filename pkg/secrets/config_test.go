// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package secrets_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/gardener-extension-registry-cache/pkg/secrets"
)

func TestSecrets(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Secrets")
}

var _ = Describe("Secrets", func() {

	DescribeTable("#TLSSecretNameForUpstream",
		func(upstream, expected string) {
			Expect(secrets.TLSSecretNameForUpstream(upstream)).To(Equal(expected))
		},
		Entry("upstream without port", "my-registry.io", "my-registry.io-tls"),
		Entry("upstream ends with port", "my-registry.io:5000", "my-registry.io-5000-tls"),
	)
})
