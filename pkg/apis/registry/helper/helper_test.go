// Copyright (c) 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helper_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"

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
		Entry("garbageCollection is nil", &registry.RegistryCache{GarbageCollection: nil}, true),
		Entry("garbageCollection.enabled is false", &registry.RegistryCache{GarbageCollection: &registry.GarbageCollection{Enabled: false}}, false),
		Entry("garbageCollection.enabled is true", &registry.RegistryCache{GarbageCollection: &registry.GarbageCollection{Enabled: true}}, true),
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
			[]registry.RegistryCache{{Upstream: "gcr.io"}, {Upstream: "quay.io"}, {Upstream: "registry.k8s.io"}},
			"docker.io",
			false, registry.RegistryCache{},
		),
		Entry("with cache with the given upstream",
			[]registry.RegistryCache{{Upstream: "gcr.io"}, {Upstream: "quay.io"}, {Upstream: "docker.io", Volume: &registry.Volume{Size: &size}}, {Upstream: "registry.k8s.io"}},
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
		Entry("volume.storageClassname is not nil", &registry.RegistryCache{Volume: &registry.Volume{StorageClassName: pointer.String("foo")}}, pointer.String("foo")),
	)
})
