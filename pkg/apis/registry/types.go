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

package registry

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegistryConfig contains information about registry caches to deploy.
type RegistryConfig struct {
	metav1.TypeMeta

	// Caches is a slice of registry caches to deploy.
	Caches []RegistryCache
}

// RegistryCache represents a registry cache to deploy.
type RegistryCache struct {
	// Upstream is the remote registry host (and optionally port) to cache.
	Upstream string
	// Size is the size of the registry cache.
	Size *resource.Quantity
	// GarbageCollection contains settings for the garbage collection of content from the cache.
	GarbageCollection *GarbageCollection
	// SecretReferenceName is the name of the reference for the Secret containing the upstream registry credentials
	SecretReferenceName *string
}

// GarbageCollection contains settings for the garbage collection of content from the cache.
type GarbageCollection struct {
	// Enabled indicates whether the garbage collection is enabled.
	Enabled bool
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegistryStatus contains information about deployed registry caches.
type RegistryStatus struct {
	metav1.TypeMeta

	// Caches is a slice of deployed registry caches.
	Caches []RegistryCacheStatus
}

// RegistryCacheStatus represents a deployed registry cache.
type RegistryCacheStatus struct {
	// Upstream is the remote registry host (and optionally port).
	Upstream string
	// Endpoint is the registry cache endpoint.
	// Example: "http://10.4.246.205:5000"
	Endpoint string
}
