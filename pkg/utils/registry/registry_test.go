// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
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

	DescribeTable("#ComputeUpstreamLabelValue",
		func(upstream, expected string) {
			actual := registryutils.ComputeUpstreamLabelValue(upstream)
			Expect(len(actual)).NotTo(BeNumerically(">", 43))
			Expect(actual).To(Equal(expected))
		},

		Entry("short upstream", "my-registry.io", "my-registry.io"),
		Entry("short upstream ends with port", "my-registry.io:5000", "my-registry.io-5000"),
		Entry("short upstream ends like a port", "my-registry.io-5000", "my-registry.io-5000"),
		Entry("long upstream", "my-very-long-registry.very-long-subdomain.io", "my-very-long-registry.very-long-subdo-2fae3"),
		Entry("long upstream ends with port", "my-very-long-registry.long-subdomain.io:8443", "my-very-long-registry.long-subdomain.-8cb9e"),
		Entry("long upstream ends like a port", "my-very-long-registry.long-subdomain.io-8443", "my-very-long-registry.long-subdomain.-e91ed"),
	)

	DescribeTable("#ComputeKubernetesResourceName",
		func(upstream, expected string) {
			Expect(registryutils.ComputeKubernetesResourceName(upstream)).To(Equal(expected))
		},
		Entry("short upstream", "my-registry.io", "registry-my-registry-io"),
		Entry("short upstream ends with port", "my-registry.io:5000", "registry-my-registry-io-5000"),
		Entry("short upstream ends like a port", "my-registry.io-5000", "registry-my-registry-io-5000"),
		Entry("long upstream", "my-very-long-registry.very-long-subdomain.io", "registry-my-very-long-registry-very-long-subdo-2fae3"),
		Entry("long upstream ends with port", "my-very-long-registry.long-subdomain.io:8443", "registry-my-very-long-registry-long-subdomain--8cb9e"),
		Entry("long upstream ends like a port", "my-very-long-registry.long-subdomain.io-8443", "registry-my-very-long-registry-long-subdomain--e91ed"),
	)
})
