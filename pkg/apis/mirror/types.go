// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package mirror

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MirrorConfig contains information about registry mirrors to configure.
type MirrorConfig struct {
	metav1.TypeMeta

	// Mirrors is a slice of registry mirrors to configure.
	Mirrors []MirrorConfiguration
}

// MirrorConfiguration represents a registry mirror.
type MirrorConfiguration struct {
	// Upstream is the remote registry host to mirror.
	// The value must be a valid DNS subdomain (RFC 1123) and optionally a port.
	Upstream string
	// Hosts are the mirror hosts to be used for the upstream.
	Hosts []MirrorHost
}

// MirrorHost represents a mirror host.
type MirrorHost struct {
	// Host is the mirror host.
	Host string
	// Capabilities are the operations a host is capable of performing.
	// This also represents the set of operations for which the mirror host may be trusted to perform.
	// The supported values are "pull" and "resolve".
	Capabilities []MirrorHostCapability
}

// MirrorHostCapability represents a mirror host capability.
type MirrorHostCapability string

const (
	// MirrorHostCapabilityPull represents the capability to fetch manifests and blobs by digest.
	MirrorHostCapabilityPull MirrorHostCapability = "pull"
	// MirrorHostCapabilityResolve represents the capability to fetch manifests by name.
	MirrorHostCapabilityResolve MirrorHostCapability = "resolve"
)
