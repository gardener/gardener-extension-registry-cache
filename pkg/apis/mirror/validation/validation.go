// Copyright (c) 2024 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror"
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

	allErrs = append(allErrs, validateUpstream(fldPath.Child("upstream"), mirror.Upstream)...)

	if len(mirror.Hosts) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("hosts"), "at least one host must be provided"))
	}

	hosts := sets.New[string]()
	for i, host := range mirror.Hosts {
		hostFldPath := fldPath.Child("hosts").Index(i)

		if !strings.HasPrefix(host.Host, "http://") && !strings.HasPrefix(host.Host, "https://") {
			allErrs = append(allErrs, field.Invalid(hostFldPath.Child("host"), host.Host, "mirror must include scheme 'http://' or 'https://'"))
		}

		if hosts.Has(host.Host) {
			allErrs = append(allErrs, field.Duplicate(hostFldPath.Child("host"), host.Host))
		} else {
			hosts.Insert(host.Host)
		}

		for _, capability := range host.Capabilities {
			if !supportedCapabilities.Has(string(capability)) {
				allErrs = append(allErrs, field.NotSupported(hostFldPath.Child("capabilities"), string(capability), sets.List(supportedCapabilities)))
			}
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
