// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registry_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	registryutils "github.com/gardener/gardener-extension-registry-cache/pkg/utils/registry"
)

func TestRegistryUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registry Utils")
}

var _ = Describe("Registry utils", func() {

	DescribeTable("#GetUpstreamURL",
		func(upstream, expected string) {
			Expect(registryutils.GetUpstreamURL(upstream)).To(Equal(expected))
		},
		Entry("upstream is docker.io", "docker.io", "https://registry-1.docker.io"),
		Entry("upstream is europe-docker.pkg.dev", "europe-docker.pkg.dev", "https://europe-docker.pkg.dev"),
		Entry("upstream is quay.io", "quay.io", "https://quay.io"),
	)
})
