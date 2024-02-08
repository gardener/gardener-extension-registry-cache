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

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/helper"
)

// ValidateRegistryConfig validates the passed configuration instance.
func ValidateRegistryConfig(config *registry.RegistryConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(config.Caches) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("caches"), "at least one cache must be provided"))
	}

	upstreams := sets.New[string]()
	for i, cache := range config.Caches {
		allErrs = append(allErrs, validateRegistryCache(cache, fldPath.Child("caches").Index(i))...)

		if upstreams.Has(cache.Upstream) {
			allErrs = append(allErrs, field.Duplicate(fldPath.Child("caches").Index(i).Child("upstream"), cache.Upstream))
		} else {
			upstreams.Insert(cache.Upstream)
		}
	}

	return allErrs
}

// ValidateRegistryConfigUpdate validates the passed configuration update.
func ValidateRegistryConfigUpdate(oldConfig, newConfig *registry.RegistryConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for i, newCache := range newConfig.Caches {
		if ok, oldCache := helper.FindCacheByUpstream(oldConfig.Caches, newCache.Upstream); ok {
			cacheFldPath := fldPath.Child("caches").Index(i)

			// We don't use the apivalidation.ValidateImmutableField func for the volume size field immutability check to be able to pass
			// string representation of it as invalid value in order to better display the invalid value.
			if !apiequality.Semantic.DeepEqual(helper.VolumeSize(&oldCache), helper.VolumeSize(&newCache)) {
				allErrs = append(allErrs, field.Invalid(cacheFldPath.Child("volume").Child("size"), helper.VolumeSize(&newCache).String(), "field is immutable"))
			}

			allErrs = append(allErrs, apivalidation.ValidateImmutableField(helper.VolumeStorageClassName(&newCache), helper.VolumeStorageClassName(&oldCache), cacheFldPath.Child("volume").Child("storageClassName"))...)

			// Mitigation for https://github.com/distribution/distribution/issues/4249
			if !helper.GarbageCollectionEnabled(&oldCache) && helper.GarbageCollectionEnabled(&newCache) {
				allErrs = append(allErrs, field.Invalid(cacheFldPath.Child("garbageCollection").Child("ttl"), newCache.GarbageCollection, "garbage collection cannot be enabled (ttl > 0) once it is disabled (ttl = 0)"))
			}
		}
	}

	return allErrs
}

func validateRegistryCache(cache registry.RegistryCache, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validateUpstream(fldPath.Child("upstream"), cache.Upstream)...)
	if cache.Volume != nil {
		if cache.Volume.Size != nil {
			allErrs = append(allErrs, validatePositiveQuantity(*cache.Volume.Size, fldPath.Child("volume", "size"))...)
		}
	}
	if cache.GarbageCollection != nil {
		if ttl := cache.GarbageCollection.TTL; ttl.Duration < 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("garbageCollection").Child("ttl"), ttl.Duration.String(), "ttl must be a non-negative duration"))
		}
	}

	return allErrs
}

func validateUpstream(fldPath *field.Path, upstream string) field.ErrorList {
	var allErrs field.ErrorList
	for _, msg := range validation.IsDNS1123Subdomain(upstream) {
		allErrs = append(allErrs, field.Invalid(fldPath, upstream, msg))
	}

	return allErrs
}

// validatePositiveQuantity validates that a Quantity is positive.
func validatePositiveQuantity(value resource.Quantity, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if value.Cmp(resource.Quantity{}) <= 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, value.String(), "must be greater than 0"))
	}
	return allErrs
}

const (
	username = "username"
	password = "password"
)

// ValidateUpstreamRegistrySecret checks whether the given Secret is immutable and contains `data.username` and `data.password` fields.
func ValidateUpstreamRegistrySecret(secret *corev1.Secret, fldPath *field.Path, secretReference string) field.ErrorList {
	var allErrors field.ErrorList

	secretRef := fmt.Sprintf("%s/%s", secret.Namespace, secret.Name)

	if secret.Immutable == nil || !*secret.Immutable {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("referenced secret %q should be immutable", secretRef)))
	}
	if len(secret.Data) != 2 {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("referenced secret %q should have only two data entries", secretRef)))
	}
	if _, ok := secret.Data[username]; !ok {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("missing %q data entry in referenced secret %q", username, secretRef)))
	}
	if _, ok := secret.Data[password]; !ok {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("missing %q data entry in referenced secret %q", password, secretRef)))
	}

	return allErrors
}
