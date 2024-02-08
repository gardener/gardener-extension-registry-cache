// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package v1alpha2

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
			Enabled: true,
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