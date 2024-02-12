// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegistryConfig contains information about registry caches to deploy.
type RegistryConfig struct {
	metav1.TypeMeta `json:",inline"`

	// Caches is a slice of registry caches to deploy.
	Caches []RegistryCache `json:"caches"`
}

// RegistryCache represents a registry cache to deploy.
type RegistryCache struct {
	// Upstream is the remote registry host to cache.
	// The value must be a valid DNS subdomain (RFC 1123).
	Upstream string `json:"upstream"`
	// Volume contains settings for the registry cache volume.
	// +optional
	Volume *Volume `json:"volume,omitempty"`
	// GarbageCollection contains settings for the garbage collection of content from the cache.
	// Defaults to enabled garbage collection.
	// +optional
	GarbageCollection *GarbageCollection `json:"garbageCollection,omitempty"`
	// SecretReferenceName is the name of the reference for the Secret containing the upstream registry credentials.
	// +optional
	SecretReferenceName *string `json:"secretReferenceName,omitempty"`
}

// Volume contains settings for the registry cache volume.
type Volume struct {
	// Size is the size of the registry cache volume.
	// Defaults to 10Gi.
	// This field is immutable.
	// +optional
	Size *resource.Quantity `json:"size,omitempty"`
	// StorageClassName is the name of the StorageClass used by the registry cache volume.
	// This field is immutable.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// GarbageCollection contains settings for the garbage collection of content from the cache.
type GarbageCollection struct {
	// Enabled indicates whether the garbage collection is enabled.
	Enabled bool `json:"enabled"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegistryStatus contains information about deployed registry caches.
type RegistryStatus struct {
	metav1.TypeMeta `json:",inline"`

	// Caches is a slice of deployed registry caches.
	Caches []RegistryCacheStatus `json:"caches"`
}

// RegistryCacheStatus represents a deployed registry cache.
type RegistryCacheStatus struct {
	// Upstream is the remote registry host.
	Upstream string `json:"upstream"`
	// Endpoint is the registry cache endpoint.
	// Example: "http://10.4.246.205:5000"
	Endpoint string `json:"endpoint"`
}
