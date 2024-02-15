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

package helper

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
)

// GarbageCollectionEnabled returns whether the garbage collection is enabled (ttl > 0) for the given cache.
func GarbageCollectionEnabled(cache *registry.RegistryCache) bool {
	return GarbageCollectionTTL(cache).Duration > 0
}

// GarbageCollectionTTL returns the time to live of a blob in the given cache.
func GarbageCollectionTTL(cache *registry.RegistryCache) metav1.Duration {
	if cache.GarbageCollection == nil {
		return registry.DefaultTTL
	}

	return cache.GarbageCollection.TTL
}

// FindCacheByUpstream finds a cache by upstream.
// The first return argument is whether the extension was found.
// The second return argument is the cache itself. An empty cache is returned if the cache is not found.
func FindCacheByUpstream(caches []registry.RegistryCache, upstream string) (bool, registry.RegistryCache) {
	for _, cache := range caches {
		if cache.Upstream == upstream {
			return true, cache
		}
	}

	return false, registry.RegistryCache{}
}

// VolumeSize returns the volume size for the given cache.
func VolumeSize(cache *registry.RegistryCache) *resource.Quantity {
	if cache.Volume == nil {
		return nil
	}

	return cache.Volume.Size
}

// VolumeStorageClassName returns the volume StorageClass name for the given cache.
func VolumeStorageClassName(cache *registry.RegistryCache) *string {
	if cache.Volume == nil {
		return nil
	}

	return cache.Volume.StorageClassName
}
