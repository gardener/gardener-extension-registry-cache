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
	// New field for containerd server configuration
	Server string `json:"server,omitempty"`
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
	// override_path is used to indicate the host's API root endpoint is defined in the URL path rather than by the API specification.
	// This may be used with non-compliant OCI registries which are missing the /v2 prefix. Defaults to false.
	// +optional
	OverridePath bool `json:"override_path"`
}

// MirrorHostCapability represents a mirror host capability.
type MirrorHostCapability string

const (
	// MirrorHostCapabilityPull represents the capability to fetch manifests and blobs by digest.
	MirrorHostCapabilityPull MirrorHostCapability = "pull"
	// MirrorHostCapabilityResolve represents the capability to fetch manifests by name.
	MirrorHostCapabilityResolve MirrorHostCapability = "resolve"
)

type MirrorHostOverridePath bool

const (
	MirrorHostOverridePathTrue MirrorHostOverridePath = true
	MirrorHostOverridePathFalse MirrorHostOverridePath = false
)
