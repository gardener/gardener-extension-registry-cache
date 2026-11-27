// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package spegel

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SpegelConfig contains information about the Spegel listening addresses of each Node.
type SpegelConfig struct {
	metav1.TypeMeta

	// RegistryPort is the port that serves the OCI registry on each Node.
	RegistryPort *int32
	// RouterPort is the port for P2P router on each Node.
	RouterPort *int32
	// MetricsPort is the metrics port on each Node.
	MetricsPort *int32
	// ResolveTags when true Spegel will resolve tags to digests.
	ResolveTags *bool
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SpegelStatus contains information about Spegel client TLS secrets.
type SpegelStatus struct {
	metav1.TypeMeta

	// CASecretName is the name of the CA bundle secret.
	CASecretName string

	// ClientTLSSecretName is the name of the Spegel client TLS secret.
	ClientTLSSecretName string
}
