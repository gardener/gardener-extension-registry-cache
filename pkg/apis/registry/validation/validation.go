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

package validation

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
)

// ValidateRegistryConfig validates the passed configuration instance.
func ValidateRegistryConfig(config *registry.RegistryConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for i, cache := range config.Caches {
		allErrs = append(allErrs, validateRegistryCache(cache, fldPath.Child("caches").Index(i))...)
	}

	return allErrs
}

func validateRegistryCache(cache registry.RegistryCache, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validateUpstream(fldPath.Child("upstream"), cache.Upstream)...)
	if size := cache.Size; size != nil && size.Sign() != 1 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("size"), size, "size must be a quantity greater than zero"))
	}

	return allErrs
}

func validateUpstream(fldPath *field.Path, upstream string) field.ErrorList {
	var allErrors field.ErrorList

	const form = "; desired format: host[:port]"
	if len(upstream) == 0 {
		allErrors = append(allErrors, field.Required(fldPath, "upstream must be provided"+form))
	}

	if strings.HasPrefix(upstream, "https://") || strings.HasPrefix(upstream, "http://") {
		allErrors = append(allErrors, field.Invalid(fldPath, upstream, "upstream must not include a scheme"+form))
	}

	return allErrors
}
