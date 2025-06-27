// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helper_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/helper"
)

func TestHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "APIs Registry Helper Suite")
}

var _ = Describe("Helpers", func() {
	size := resource.MustParse("5Gi")

	DescribeTable("#GarbageCollectionEnabled",
		func(cache *registry.RegistryCache, expected bool) {
			Expect(helper.GarbageCollectionEnabled(cache)).To(Equal(expected))
		},
		Entry("garbageCollection is nil",
			&registry.RegistryCache{GarbageCollection: nil},
			true,
		),
		Entry("garbageCollection.ttl is zero",
			&registry.RegistryCache{GarbageCollection: &registry.GarbageCollection{TTL: metav1.Duration{Duration: 0}}},
			false,
		),
		Entry("garbageCollection.ttl is a positive duration",
			&registry.RegistryCache{GarbageCollection: &registry.GarbageCollection{TTL: metav1.Duration{Duration: 30 * 24 * time.Hour}}},
			true,
		),
	)

	DescribeTable("#GarbageCollectionTTL",
		func(cache *registry.RegistryCache, expected metav1.Duration) {
			Expect(helper.GarbageCollectionTTL(cache)).To(Equal(expected))
		},
		Entry("garbageCollection is nil",
			&registry.RegistryCache{GarbageCollection: nil},
			metav1.Duration{Duration: 7 * 24 * time.Hour},
		),
		Entry("garbageCollection.ttl is zero",
			&registry.RegistryCache{GarbageCollection: &registry.GarbageCollection{TTL: metav1.Duration{Duration: 0}}},
			metav1.Duration{Duration: 0},
		),
		Entry("garbageCollection.ttl is a positive duration",
			&registry.RegistryCache{GarbageCollection: &registry.GarbageCollection{TTL: metav1.Duration{Duration: 30 * 24 * time.Hour}}},
			metav1.Duration{Duration: 30 * 24 * time.Hour},
		),
	)

	DescribeTable("#FindCacheByUpstream",
		func(caches []registry.RegistryCache, upstream string, expectedOk bool, expectedCache registry.RegistryCache) {
			ok, cache := helper.FindCacheByUpstream(caches, upstream)
			Expect(ok).To(Equal(expectedOk))
			Expect(cache).To(Equal(expectedCache))
		},
		Entry("caches is nil",
			nil,
			"docker.io",
			false, registry.RegistryCache{},
		),
		Entry("caches is empty",
			[]registry.RegistryCache{},
			"docker.io",
			false, registry.RegistryCache{},
		),
		Entry("no cache with the given upstream",
			[]registry.RegistryCache{{Upstream: "europe-docker.pkg.dev"}, {Upstream: "quay.io"}, {Upstream: "registry.k8s.io"}},
			"docker.io",
			false, registry.RegistryCache{},
		),
		Entry("with cache with the given upstream",
			[]registry.RegistryCache{{Upstream: "europe-docker.pkg.dev"}, {Upstream: "quay.io"}, {Upstream: "docker.io", Volume: &registry.Volume{Size: &size}}, {Upstream: "registry.k8s.io"}},
			"docker.io",
			true, registry.RegistryCache{Upstream: "docker.io", Volume: &registry.Volume{Size: &size}},
		),
	)

	DescribeTable("#VolumeSize",
		func(cache *registry.RegistryCache, expected *resource.Quantity) {
			Expect(helper.VolumeSize(cache)).To(Equal(expected))
		},
		Entry("volume is nil", &registry.RegistryCache{Volume: nil}, nil),
		Entry("volume.size is not nil", &registry.RegistryCache{Volume: &registry.Volume{Size: &size}}, &size),
	)

	DescribeTable("#VolumeStorageClassName",
		func(cache *registry.RegistryCache, expected *string) {
			Expect(helper.VolumeStorageClassName(cache)).To(Equal(expected))
		},
		Entry("volume is nil", &registry.RegistryCache{Volume: nil}, nil),
		Entry("volume.storageClassname is not nil", &registry.RegistryCache{Volume: &registry.Volume{StorageClassName: ptr.To("foo")}}, ptr.To("foo")),
	)

	DescribeTable("#TLSEnabled",
		func(cache *registry.RegistryCache, expected bool) {
			Expect(helper.TLSEnabled(cache)).To(Equal(expected))
		},
		Entry("http is nil", &registry.RegistryCache{HTTP: nil}, true),
		Entry("http.tls is false", &registry.RegistryCache{HTTP: &registry.HTTP{TLS: false}}, false),
		Entry("http.tls is true", &registry.RegistryCache{HTTP: &registry.HTTP{TLS: true}}, true),
	)

	DescribeTable("#HighAvailabilityEnabled",
		func(cache *registry.RegistryCache, expected bool) {
			Expect(helper.HighAvailabilityEnabled(cache)).To(Equal(expected))
		},
		Entry("highAvailability is nil", &registry.RegistryCache{HighAvailability: nil}, false),
		Entry("highAvailability.enabled is false", &registry.RegistryCache{HighAvailability: &registry.HighAvailability{Enabled: false}}, false),
		Entry("highAvailability.enabled is true", &registry.RegistryCache{HighAvailability: &registry.HighAvailability{Enabled: true}}, true),
	)
})
