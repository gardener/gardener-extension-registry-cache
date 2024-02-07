// Copyright 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package v1alpha1

import (
	conversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/utils/ptr"

	registry "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
)

// Convert_v1alpha1_RegistryCache_To_registry_RegistryCache converts from v1alpha1.RegistryCache to registry.RegistryCache.
func Convert_v1alpha1_RegistryCache_To_registry_RegistryCache(in *RegistryCache, out *registry.RegistryCache, s conversion.Scope) error {
	if err := autoConvert_v1alpha1_RegistryCache_To_registry_RegistryCache(in, out, s); err != nil {
		return err
	}

	out.Volume = &registry.Volume{
		Size: in.Size,
		// In v1alpha1 the StorageClass name was not configurable and the registry-cache extension assumed the StorageClass name to be "default".
		// To preserve backwards-compatibility we set the StorageClassName field to "default".
		// There are already many StatefulSets created according to the v1alpha1 RegistryConfig and for the the registry-cache extension set the
		// StorageClass name in the StatefulSet to "default". The corresponding StatetulSet field is immutable.
		StorageClassName: ptr.To("default"),
	}

	return nil
}

// Convert_registry_RegistryCache_To_v1alpha1_RegistryCache converts from registry.RegistryCache to v1alpha1.RegistryCache.
func Convert_registry_RegistryCache_To_v1alpha1_RegistryCache(in *registry.RegistryCache, out *RegistryCache, s conversion.Scope) error {
	if err := autoConvert_registry_RegistryCache_To_v1alpha1_RegistryCache(in, out, s); err != nil {
		return err
	}

	if in.Volume != nil {
		out.Size = in.Volume.Size
	}

	return nil
}
