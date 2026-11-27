// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/spegel"
)

// ValidateSpegelConfig validates the passed configuration instance.
func ValidateSpegelConfig(spegelConfig *spegel.SpegelConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validatePort(spegelConfig.RegistryPort, fldPath.Child("registryPort"), "registry port")...)
	allErrs = append(allErrs, validatePort(spegelConfig.RouterPort, fldPath.Child("routerPort"), "router port")...)
	allErrs = append(allErrs, validatePort(spegelConfig.MetricsPort, fldPath.Child("metricsPort"), "metrics port")...)

	allErrs = append(allErrs, validateDistinctPorts(spegelConfig, fldPath)...)

	return allErrs
}

// validatePort validates that the given port is provided and is a valid port number.
func validatePort(port *int32, fldPath *field.Path, name string) field.ErrorList {
	allErrs := field.ErrorList{}

	if port == nil {
		allErrs = append(allErrs, field.Required(fldPath, name+" must be provided"))
		return allErrs
	}
	for _, msg := range validation.IsValidPortNum(int(*port)) {
		allErrs = append(allErrs, field.Invalid(fldPath, *port, msg))
	}

	return allErrs
}

// validateDistinctPorts validates that the registry, router and metrics ports do not collide with each other.
func validateDistinctPorts(spegelConfig *spegel.SpegelConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	ports := map[int32]*field.Path{}
	for _, p := range []struct {
		port    *int32
		fldPath *field.Path
	}{
		{spegelConfig.RegistryPort, fldPath.Child("registryPort")},
		{spegelConfig.RouterPort, fldPath.Child("routerPort")},
		{spegelConfig.MetricsPort, fldPath.Child("metricsPort")},
	} {
		if p.port == nil {
			continue
		}
		if _, ok := ports[*p.port]; ok {
			allErrs = append(allErrs, field.Invalid(p.fldPath, *p.port, "port must not be equal to the other Spegel ports"))
		} else {
			ports[*p.port] = p.fldPath
		}
	}

	return allErrs
}
