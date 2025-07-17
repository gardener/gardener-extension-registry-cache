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
	serviceNameSuffixes := sets.New[string]()
	for i, cache := range config.Caches {
		allErrs = append(allErrs, validateRegistryCache(cache, fldPath.Child("caches").Index(i))...)

		if upstreams.Has(cache.Upstream) {
			allErrs = append(allErrs, field.Duplicate(fldPath.Child("caches").Index(i).Child("upstream"), cache.Upstream))
		} else {
			upstreams.Insert(cache.Upstream)
		}
	}

	for i, cache := range config.Caches {
		if cache.ServiceNameSuffix != nil {
			if serviceNameSuffixes.Has(*cache.ServiceNameSuffix) {
				allErrs = append(allErrs, field.Duplicate(fldPath.Child("caches").Index(i).Child("serviceNameSuffix"), *cache.ServiceNameSuffix))
			} else {
				serviceNameSuffixes.Insert(*cache.ServiceNameSuffix)
			}

			if cache.Upstream != *cache.ServiceNameSuffix && upstreams.Has(*cache.ServiceNameSuffix) {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("caches").Index(i).Child("serviceNameSuffix"), *cache.ServiceNameSuffix, "cannot collide with upstream"))
			}
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
	if cache.ServiceNameSuffix != nil {
		allErrs = append(allErrs, ValidateServiceNameSuffix(fldPath.Child("serviceNameSuffix"), *cache.ServiceNameSuffix)...)
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

// A label value length and a resource name length limits are 63 chars.
// The cache resources name have prefix 'registry-', thus the label value length is limited to 54.
const serviceNameSuffixValueLimit = 54

// ValidateServiceNameSuffix validates that serviceNameSuffix is valid DNS subdomain (RFC 1123) and not longer than 54 characters.
func ValidateServiceNameSuffix(fldPath *field.Path, serviceNameSuffix string) field.ErrorList {
	var allErrs field.ErrorList
	for _, msg := range validation.IsDNS1123Label(serviceNameSuffix) {
		allErrs = append(allErrs, field.Invalid(fldPath, serviceNameSuffix, msg))
	}

	if len(serviceNameSuffix) > serviceNameSuffixValueLimit {
		allErrs = append(allErrs, field.Invalid(fldPath, serviceNameSuffix, fmt.Sprintf("cannot be longer than %d characters", serviceNameSuffixValueLimit)))
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
	usernameKey = "username"
	passwordKey = "password"
)

// ValidateUpstreamRegistrySecret checks whether the given Secret is immutable and contains `data.username` and `data.password` fields.
func ValidateUpstreamRegistrySecret(secret *corev1.Secret, fldPath *field.Path, secretReferenceName string) field.ErrorList {
	var allErrors field.ErrorList

	secretKey := fmt.Sprintf("%s/%s", secret.Namespace, secret.Name)

	if secret.Immutable == nil || !*secret.Immutable {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReferenceName, fmt.Sprintf("referenced secret %q should be immutable", secretKey)))
	}
	if len(secret.Data) != 2 {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReferenceName, fmt.Sprintf("referenced secret %q should have only two data entries", secretKey)))
	}
	if username, ok := secret.Data[usernameKey]; ok {
		if len(bytes.TrimSpace(username)) == 0 {
			allErrors = append(allErrors, field.Invalid(fldPath, secretReferenceName, fmt.Sprintf("data entry %q in referenced secret %q is empty", usernameKey, secretKey)))
		}
		if bytes.ContainsFunc(username, unicode.IsSpace) {
			allErrors = append(allErrors, field.Invalid(fldPath, secretReferenceName, fmt.Sprintf("data entry %q in referenced secret %q contains whitespace", usernameKey, secretKey)))
		}
	} else {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReferenceName, fmt.Sprintf("missing %q data entry in referenced secret %q", usernameKey, secretKey)))
	}
	if password, ok := secret.Data[passwordKey]; ok {
		if len(bytes.TrimSpace(password)) == 0 {
			allErrors = append(allErrors, field.Invalid(fldPath, secretReferenceName, fmt.Sprintf("data entry %q in referenced secret %q is empty", passwordKey, secretKey)))
		}
	} else {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReferenceName, fmt.Sprintf("missing %q data entry in referenced secret %q", passwordKey, secretKey)))
	}

	// validate ServiceAccount json
	if username, ok := secret.Data[usernameKey]; ok && string(username) == "_json_key" {
		if password, ok := secret.Data[passwordKey]; ok {
			allErrors = append(allErrors, validateServiceAccountJSON(password, fldPath, secretReferenceName, secretKey)...)
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

var serviceAccountAllowedFields = sets.New(
	"type",
	"project_id",
	"client_email",
	"universe_domain",
	"auth_uri",
	"auth_provider_x509_cert_url",
	"client_x509_cert_url",
	"client_id",
	"private_key_id",
	"private_key",
	"token_uri",
)

func validateServiceAccountJSON(serviceAccountJSON []byte, fldPath *field.Path, secretReferenceName, secretKey string) field.ErrorList {
	var allErrors field.ErrorList

	serviceAccountMap := map[string]string{}
	if err := json.Unmarshal(serviceAccountJSON, &serviceAccountMap); err != nil {
		allErrors = append(allErrors, field.Invalid(fldPath, secretReferenceName, fmt.Sprintf("failed to unmarshal ServiceAccount json from password data entry in referenced secret %q: %v", secretKey, err)))
	}

	for fld := range serviceAccountMap {
		if !serviceAccountAllowedFields.Has(fld) {
			allErrors = append(allErrors, field.Invalid(fldPath, secretReferenceName, fmt.Sprintf("forbidden ServiceAccount field %q present in password data entry in referenced secret %q", fld, secretKey)))
		}
	}

	return allErrors
}
