// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

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
	if server := mirror.Server; server != "" {
	    allErrs = append(allErrs, registryvalidation.ValidateURL(fldPath.Child("server"), mirror.Server)...)
        }

	if len(mirror.Hosts) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("hosts"), "at least one host must be provided"))
	}

	hosts := sets.New[string]()
	for i, host := range mirror.Hosts {
		hostFldPath := fldPath.Child("hosts").Index(i)

		allErrs = append(allErrs, registryvalidation.ValidateURL(hostFldPath.Child("host"), host.Host)...)

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
