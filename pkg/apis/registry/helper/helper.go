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
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
)

// FindCacheByUpstream finds a cache by upstream.
// The first return argument is whether the extension was found.
// The third return arguement is the cache itself. An empty cache is returned if the cache is not found.
func FindCacheByUpstream(caches []registry.RegistryCache, upstream string) (bool, registry.RegistryCache) {
	for _, cache := range caches {
		if cache.Upstream == upstream {
			return true, cache
		}
	}

	return false, registry.RegistryCache{}
}
