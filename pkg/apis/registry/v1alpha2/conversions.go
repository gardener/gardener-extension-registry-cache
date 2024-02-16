// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"

	registry "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
)

// Convert_v1alpha2_GarbageCollection_To_registry_GarbageCollection converts from v1alpha2.GarbageCollection to registry.GarbageCollection.
func Convert_v1alpha2_GarbageCollection_To_registry_GarbageCollection(in *GarbageCollection, out *registry.GarbageCollection, _ conversion.Scope) error {
	if in.Enabled {
		out.TTL = registry.DefaultTTL
	} else {
		out.TTL = metav1.Duration{Duration: 0}
	}

	return nil
}

// Convert_registry_GarbageCollection_To_v1alpha2_GarbageCollection converts from registry.GarbageCollection to v1alpha2.GarbageCollection.
func Convert_registry_GarbageCollection_To_v1alpha2_GarbageCollection(in *registry.GarbageCollection, out *GarbageCollection, _ conversion.Scope) error {
	if in.TTL.Duration > 0 {
		out.Enabled = true
	} else {
		out.Enabled = false
	}

	return nil
}
