// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"

	"github.com/gardener/gardener/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror"
	registryvalidation "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/validation"
)

var supportedCapabilities = sets.New[string](
	string(mirror.MirrorHostCapabilityPull),
	string(mirror.MirrorHostCapabilityResolve),
)

// ValidateMirrorConfig validates the passed configuration instance.
func ValidateMirrorConfig(mirrorConfig *mirror.MirrorConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(mirrorConfig.Mirrors) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("mirrors"), "at least one mirror must be provided"))
	}

	upstreams := sets.New[string]()
	for i, mirror := range mirrorConfig.Mirrors {
		configFldPath := fldPath.Child("mirrors").Index(i)

		allErrs = append(allErrs, validateMirrorConfiguration(mirror, configFldPath)...)

		if upstreams.Has(mirror.Upstream) {
			allErrs = append(allErrs, field.Duplicate(configFldPath.Child("upstream"), mirror.Upstream))
		} else {
			upstreams.Insert(mirror.Upstream)
		}
	}

	return allErrs
}

func validateMirrorConfiguration(mirror mirror.MirrorConfiguration, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, registryvalidation.ValidateUpstream(fldPath.Child("upstream"), mirror.Upstream)...)

	if len(mirror.Hosts) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("hosts"), "at least one host must be provided"))
	}

	hosts := sets.New[string]()
	for i, host := range mirror.Hosts {
		hostFldPath := fldPath.Child("hosts").Index(i)

		allErrs = append(allErrs, registryvalidation.ValidateURL(hostFldPath.Child("host"), host.Host, true)...)

		if hosts.Has(host.Host) {
			allErrs = append(allErrs, field.Duplicate(hostFldPath.Child("host"), host.Host))
		} else {
			hosts.Insert(host.Host)
		}

		allErrs = append(allErrs, validateCapabilities(hostFldPath.Child("capabilities"), host.Capabilities)...)
	}

	return allErrs
}

func validateCapabilities(fldPath *field.Path, capabilities []mirror.MirrorHostCapability) field.ErrorList {
	var allErrs field.ErrorList

	capabilitiesFound := sets.New[string]()
	for i, capability := range capabilities {
		capabilityAsString := string(capability)

		if !supportedCapabilities.Has(capabilityAsString) {
			allErrs = append(allErrs, field.NotSupported(fldPath, capabilityAsString, sets.List(supportedCapabilities)))
		}

		if capabilitiesFound.Has(capabilityAsString) {
			allErrs = append(allErrs, field.Duplicate(fldPath.Index(i), capabilityAsString))
		} else {
			capabilitiesFound.Insert(capabilityAsString)
		}
	}

	return allErrs
}

// ValidateMirrorHostCABundleSecret checks whether the given Secret is immutable and contains a valid PEM-encoded certificate.
func ValidateMirrorHostCABundleSecret(secret *corev1.Secret, fldPath *field.Path, caBundleSecretReferenceName string) field.ErrorList {
	const caBundleKey = "bundle.crt"

	var (
		allErrs   field.ErrorList
		secretKey = fmt.Sprintf("%s/%s", secret.Namespace, secret.Name)
	)

	if !ptr.Deref(secret.Immutable, false) {
		allErrs = append(allErrs, field.Invalid(fldPath, caBundleSecretReferenceName, fmt.Sprintf("the referenced CA bundle secret %q should be immutable", secretKey)))
	}

	caBundle, ok := secret.Data[caBundleKey]
	if !ok {
		allErrs = append(allErrs, field.Invalid(fldPath, caBundleSecretReferenceName, fmt.Sprintf("missing %q data entry in the referenced CA bundle secret %q", caBundleKey, secretKey)))
	} else {
		if _, err := utils.DecodeCertificate(caBundle); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath, caBundleSecretReferenceName, "the CA bundle is not a valid PEM-encoded certificate"))
		}
	}

	if len(secret.Data) > 1 {
		allErrs = append(allErrs, field.Invalid(fldPath, caBundleSecretReferenceName, fmt.Sprintf("the referenced CA bundle secret %q should only have a single data entry with key %q", secretKey, caBundleKey)))
	}

	return allErrs
}
