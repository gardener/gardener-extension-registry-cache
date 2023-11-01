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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/helper"
)

// ValidateRegistryConfigUpdate validates the passed configuration update.
func ValidateRegistryConfigUpdate(oldConfig, newConfig *registry.RegistryConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for i, newCache := range newConfig.Caches {
		if ok, oldCache := helper.FindCacheByUpstream(oldConfig.Caches, newCache.Upstream); ok {
			if !apiequality.Semantic.DeepEqual(oldCache.Size, newCache.Size) {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("caches").Index(i).Child("size"), newCache.Size.String(), "field is immutable"))
			}
		}
	}

	return allErrs
}

// ValidateRegistryCache validates the passed registry cache instance.
func ValidateRegistryCache(cache registry.RegistryCache, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validateUpstream(fldPath.Child("upstream"), cache.Upstream)...)
	if cache.Size != nil {
		allErrs = append(allErrs, validatePositiveQuantity(*cache.Size, fldPath.Child("size"))...)
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

// validatePositiveQuantity validates that a Quantity is positive.
func validatePositiveQuantity(value resource.Quantity, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if value.Cmp(resource.Quantity{}) <= 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, value.String(), "must be greater than 0"))
	}
	return allErrs
}

// ValidateUpstreamRepositorySecret checks whether the given secret is immutable and contains `data.username` and `data.password` fields.
func ValidateUpstreamRepositorySecret(secret *corev1.Secret, fldPath *field.Path, secretReference string) field.ErrorList {
	var allErrors field.ErrorList

	secretRef := fmt.Sprintf("%s/%s", secret.Namespace, secret.Name)

	if secret.Immutable == nil || !*secret.Immutable {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("referenced secret %q should be immutable", secretRef)))
	}
	if len(secret.Data) != 2 {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("referenced secret %q should have only two data entries", secretRef)))
	}
	const username = "username"
	if _, ok := secret.Data[username]; !ok {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("missing %q data entry in referenced secret %q", username, secretRef)))
	}
	const password = "password"
	if _, ok := secret.Data[password]; !ok {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("missing %q data entry in referenced secret %q", password, secretRef)))
	}

	return allErrors
}
