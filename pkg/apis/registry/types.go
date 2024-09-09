// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"time"

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
	// Upstream is the remote registry host to cache.
	// The value must be a valid DNS subdomain (RFC 1123) and optionally a port.
	Upstream string
	// RemoteURL is the remote registry URL. The format must be `<scheme><host>[:<port>]` where
	// `<scheme>` is `https://` or `http://` and `<host>[:<port>]` corresponds to the Upstream
	//
	// If defined, the value is set as `proxy.remoteurl` in the registry [configuration](https://github.com/distribution/distribution/blob/main/docs/content/recipes/mirror.md#configure-the-cache)
	// and in containerd configuration as `server` field in [hosts.toml](https://github.com/containerd/containerd/blob/main/docs/hosts.md#server-field) file.
	RemoteURL *string
	// Volume contains settings for the registry cache volume.
	Volume *Volume
	// GarbageCollection contains settings for the garbage collection of content from the cache.
	GarbageCollection *GarbageCollection
	// SecretReferenceName is the name of the reference for the Secret containing the upstream registry credentials
	SecretReferenceName *string
}

// Volume contains settings for the registry cache volume.
type Volume struct {
	// Size is the size of the registry cache volume.
	// Defaults to 10Gi.
	// This field is immutable.
	Size *resource.Quantity
	// StorageClassName is the name of the StorageClass used by the registry cache volume.
	// This field is immutable.
	StorageClassName *string
}

// GarbageCollection contains settings for the garbage collection of content from the cache.
type GarbageCollection struct {
	// TTL is the time to live of a blob in the cache.
	// Set to 0s to disable the garbage collection.
	TTL metav1.Duration
}

var (
	// DefaultTTL is the default time to live of a blob in the cache.
	DefaultTTL = metav1.Duration{Duration: 7 * 24 * time.Hour}
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegistryStatus contains information about deployed registry caches.
type RegistryStatus struct {
	metav1.TypeMeta

	// CASecretName is the name of the CA secret
	CASecretName string
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
	// RemoteURL is the remote registry URL.
	RemoteURL string
}
