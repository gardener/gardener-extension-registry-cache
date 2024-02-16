// Copyright 2024 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
