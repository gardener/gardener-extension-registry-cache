// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SpegelConfig contains information about the Spegel listening addresses of each Node.
type SpegelConfig struct {
	metav1.TypeMeta `json:",inline"`

	// RegistryPort is the port that serves the OCI registry on each Node.
	// `registryPort` should be a valid port number (1-65535, inclusive).
	// Defaults to 15500.
	// +optional
	RegistryPort *int32 `json:"registryPort,omitempty"`
	// RouterPort is the port for P2P router on each Node.
	// `routerPort` should be a valid port number (1-65535, inclusive).
	// Defaults to 15501.
	// +optional
	RouterPort *int32 `json:"routerPort,omitempty"`
	// MetricsPort is the metrics port on each Node.
	// `metricsPort` should be a valid port number (1-65535, inclusive).
	// Defaults to 19090.
	// +optional
	MetricsPort *int32 `json:"metricsPort,omitempty"`
	// ResolveTags when true Spegel will resolve tags to digests.
	// Defaults to true.
	// +optional
	ResolveTags *bool `json:"resolveTags,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SpegelStatus contains information about Spegel client TLS secrets.
type SpegelStatus struct {
	metav1.TypeMeta `json:",inline"`

	// CASecretName is the name of the CA bundle secret.
	CASecretName string `json:"caSecretName"`

	// ClientTLSSecretName is the name of the Spegel client TLS secret.
	ClientTLSSecretName string `json:"clientTLSSecretName"`
}
