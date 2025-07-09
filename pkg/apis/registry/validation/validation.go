// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"

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

	allErrs = append(allErrs, ValidateUpstream(fldPath.Child("upstream"), cache.Upstream)...)
	if cache.RemoteURL != nil {
		allErrs = append(allErrs, ValidateURL(fldPath.Child("remoteURL"), *cache.RemoteURL)...)
	}
	if cache.Volume != nil {
		if cache.Volume.Size != nil {
			allErrs = append(allErrs, validatePositiveQuantity(*cache.Volume.Size, fldPath.Child("volume", "size"))...)
		}
		if cache.Volume.StorageClassName != nil {
			for _, msg := range apivalidation.NameIsDNSSubdomain(*cache.Volume.StorageClassName, false) {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("volume", "storageClassName"), *cache.Volume.StorageClassName, msg))
			}
		}
	}
	if cache.GarbageCollection != nil {
		if ttl := cache.GarbageCollection.TTL; ttl.Duration < 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("garbageCollection").Child("ttl"), ttl.Duration.String(), "ttl must be a non-negative duration"))
		}
	}
	if cache.Proxy != nil {
		if cache.Proxy.HTTPProxy != nil {
			allErrs = append(allErrs, ValidateURL(fldPath.Child("proxy").Child("httpProxy"), *cache.Proxy.HTTPProxy)...)
		}
		if cache.Proxy.HTTPSProxy != nil {
			allErrs = append(allErrs, ValidateURL(fldPath.Child("proxy").Child("httpsProxy"), *cache.Proxy.HTTPSProxy)...)
		}
	}

	return allErrs
}

// ValidateUpstream validates that upstream is valid DNS subdomain (RFC 1123) and optionally a port.
func ValidateUpstream(fldPath *field.Path, upstream string) field.ErrorList {
	var allErrs field.ErrorList
	for _, msg := range validateHostPort(upstream) {
		allErrs = append(allErrs, field.Invalid(fldPath, upstream, msg))
	}

	return allErrs
}

var digitsRegex = regexp.MustCompile(`^\d+$`)
var portRegexp = regexp.MustCompile(`^([1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])$`)

// validateHostPort check that host and optional port format is `<host>[:<port>]`
func validateHostPort(hostPort string) []string {
	var errs []string
	host := hostPort
	if index := strings.LastIndexByte(hostPort, ':'); index != -1 {
		port := hostPort[index+1:]
		if digitsRegex.MatchString(port) {
			host = hostPort[:index]
			if !portRegexp.MatchString(port) {
				errs = append(errs, fmt.Sprintf("port '%s' is not valid, valid port must be in the range [1, 65535]", port))
			}
		}
	}
	return append(errs, validation.IsDNS1123Subdomain(host)...)
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
	if user, ok := secret.Data[username]; ok {
		if len(bytes.TrimSpace(user)) == 0 {
			allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("data entry %q in referenced secret %q is empty", username, secretRef)))
		}
		if bytes.ContainsFunc(user, unicode.IsSpace) {
			allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("data entry %q in referenced secret %q contains white spaces", username, secretRef)))
		}
	} else {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("missing %q data entry in referenced secret %q", username, secretRef)))
	}
	if pass, ok := secret.Data[password]; ok {
		if len(bytes.TrimSpace(pass)) == 0 {
			allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("data entry %q in referenced secret %q is empty", password, secretRef)))
		}
	} else {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("missing %q data entry in referenced secret %q", password, secretRef)))
	}

	// validate ServiceAccount json
	if user, ok := secret.Data[username]; ok && string(user) == "_json_key" {
		if pwd, ok := secret.Data[password]; ok {
			allErrors = append(allErrors, validateServiceAccountJson(pwd, fldPath, secretReference, secretRef)...)
		}
	}

	return allErrors
}

// ValidateURL validates that URL format is `<scheme><host>[:<port>]` where `<scheme>` is 'https://' or 'http://',
// `<host>` is valid DNS subdomain (RFC 1123) and optional `<port>` is in range [1,65535].
func ValidateURL(fldPath *field.Path, url string) field.ErrorList {
	var allErrs field.ErrorList
	var scheme string
	host := url
	index := strings.Index(url, "://")
	if index != -1 {
		scheme = url[:index]
		host = url[index+len("://"):]
	}
	if scheme != "https" && scheme != "http" {
		allErrs = append(allErrs, field.Invalid(fldPath, url, "url must start with 'http://' or 'https://' scheme"))
	}
	for _, msg := range validateHostPort(host) {
		allErrs = append(allErrs, field.Invalid(fldPath, url, msg))
	}

	return allErrs
}

var serviceAccountAllowedFields = map[string]struct{}{
	"type":                        {},
	"project_id":                  {},
	"client_email":                {},
	"universe_domain":             {},
	"auth_uri":                    {},
	"auth_provider_x509_cert_url": {},
	"client_x509_cert_url":        {},
	"client_id":                   {},
	"private_key_id":              {},
	"private_key":                 {},
	"token_uri":                   {},
}

func validateServiceAccountJson(serviceAccountJSON []byte, fldPath *field.Path, secretReference, secretRef string) field.ErrorList {
	var allErrors field.ErrorList

	serviceAccountMap := map[string]string{}
	if err := json.Unmarshal(serviceAccountJSON, &serviceAccountMap); err != nil {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("failed to unmarshal ServiceAccount json from password data entry in referenced secret %q: %v", secretRef, err)))
	}

	for fld := range serviceAccountMap {
		if _, ok := serviceAccountAllowedFields[fld]; !ok {
			allErrors = append(allErrors, field.Invalid(fldPath, secretReference, fmt.Sprintf("forbidden ServiceAccount field %q present in password data entry in referenced secret %q", fld, secretRef)))
		}
	}

	return allErrors
}
