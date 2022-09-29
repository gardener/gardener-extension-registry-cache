// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package validator

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	api "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
)

func decodeRegistryConfig(decoder runtime.Decoder, config *runtime.RawExtension, fldPath *field.Path) (*api.RegistryConfig, error) {
	if config == nil {
		return nil, field.Required(fldPath, "Registry configuration is required when using the gardener-extension-registry-cache")
	}

	registryConfig := &api.RegistryConfig{}
	if err := runtime.DecodeInto(decoder, config.Raw, registryConfig); err != nil {
		return nil, err
	}

	return registryConfig, nil
}
