// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MirrorConfig contains information about registry mirrors to configure.
type MirrorConfig struct {
	metav1.TypeMeta `json:",inline"`

	// Mirrors is a slice of registry mirrors to configure.
	Mirrors []MirrorConfiguration `json:"mirrors"`
}

// MirrorConfiguration represents a registry mirror.
type MirrorConfiguration struct {
	// Upstream is the remote registry host to mirror.
	// The value must be a valid DNS subdomain (RFC 1123) and optionally a port.
	Upstream string `json:"upstream"`
	// Hosts are the mirror hosts to be used for the upstream.
	Hosts []MirrorHost `json:"hosts"`
}

// MirrorHost represents a mirror host.
type MirrorHost struct {
	// Host is the mirror host.
	Host string `json:"host"`
	// Capabilities are the operations a host is capable of performing.
	// This also represents the set of operations for which the mirror host may be trusted to perform.
	// The supported values are "pull" and "resolve".
	// Defaults to ["pull"].
	// +optional
	Capabilities []MirrorHostCapability `json:"capabilities"`
	// CABundleSecretReferenceName is the reference name for a Secret containing a PEM-encoded certificate authority bundle.
	// The CA bundle is used to verify the TLS certificate of the mirror host.
	// The referenced secret must be immutable and must have a data key `bundle.crt`.
	// +optional
	CABundleSecretReferenceName *string `json:"caBundleSecretReferenceName"`
	// OverridePath represents the `override_path` field in the [hosts.toml](https://github.com/containerd/containerd/blob/main/docs/hosts.md#override_path-field)
	// file for containerd hosts configuration.
	// Should be set to `true` only for non-compliant OCI registries which are missing the `/v2` prefix, and the API root endpoint is defined in the host URL path.
	// +optional
	OverridePath *bool `json:"overridePath,omitempty"`
}

// MirrorHostCapability represents a mirror host capability.
type MirrorHostCapability string

const (
	// MirrorHostCapabilityPull represents the capability to fetch manifests and blobs by digest.
	MirrorHostCapabilityPull MirrorHostCapability = "pull"
	// MirrorHostCapabilityResolve represents the capability to fetch manifests by name.
	MirrorHostCapabilityResolve MirrorHostCapability = "resolve"
)
