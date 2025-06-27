// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha3

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

// SetDefaults_RegistryCache sets the defaults for a RegistryCache.
func SetDefaults_RegistryCache(cache *RegistryCache) {
	if cache.Volume == nil {
		cache.Volume = &Volume{}
	}

	if cache.GarbageCollection == nil {
		cache.GarbageCollection = &GarbageCollection{
			TTL: DefaultTTL,
		}
	}

	if cache.HTTP == nil {
		cache.HTTP = &HTTP{
			TLS: true,
		}
	}
}

// SetDefaults_Volume sets the defaults for a Volume.
func SetDefaults_Volume(volume *Volume) {
	if volume.Size == nil {
		defaultCacheSize := resource.MustParse("10Gi")
		volume.Size = &defaultCacheSize
	}
}
