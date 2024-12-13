// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package mirror

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-registry-cache/pkg/admission/validator/helper"
	mirrorapi "github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror/validation"
	cacheapi "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
)

type shoot struct {
	decoder runtime.Decoder
}

// NewShootValidator returns a new instance of a shoot validator that validates:
// - the registry-mirror providerConfig
// - the registry-mirror providerConfig against registry-cache providerConfig (if there is any)
func NewShootValidator(decoder runtime.Decoder) extensionswebhook.Validator {
	return &shoot{
		decoder: decoder,
	}
}

func (s *shoot) Validate(_ context.Context, newObj, _ client.Object) error {
	shoot, ok := newObj.(*core.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", newObj)
	}

	i, mirrorExt := helper.FindExtension(shoot.Spec.Extensions, "registry-mirror")
	if i == -1 {
		return nil
	}

	for _, worker := range shoot.Spec.Provider.Workers {
		if worker.CRI.Name != "containerd" {
			return fmt.Errorf("container runtime needs to be containerd when the registry-mirror extension is enabled")
		}
	}

	providerConfigPath := field.NewPath("spec", "extensions").Index(i).Child("providerConfig")
	if mirrorExt.ProviderConfig == nil {
		return field.Required(providerConfigPath, "providerConfig is required for the registry-mirror extension")
	}

	mirrorConfig := &mirrorapi.MirrorConfig{}
	if err := runtime.DecodeInto(s.decoder, mirrorExt.ProviderConfig.Raw, mirrorConfig); err != nil {
		return fmt.Errorf("failed to decode providerConfig: %w", err)
	}

	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validation.ValidateMirrorConfig(mirrorConfig, providerConfigPath)...)

	j, cacheExt := helper.FindExtension(shoot.Spec.Extensions, "registry-cache")
	if j != -1 {
		if cacheExt.ProviderConfig == nil {
			return fmt.Errorf("providerConfig is not available for registry-cache extension")
		}

		cacheRegistryConfig := &cacheapi.RegistryConfig{}
		if err := runtime.DecodeInto(s.decoder, cacheExt.ProviderConfig.Raw, cacheRegistryConfig); err != nil {
			return fmt.Errorf("failed to decode providerConfig: %w", err)
		}

		allErrs = append(allErrs, validateMirrorConfigAgainstRegistryCache(mirrorConfig, cacheRegistryConfig, providerConfigPath)...)

	}

	return allErrs.ToAggregate()
}

func validateMirrorConfigAgainstRegistryCache(mirrorConfig *mirrorapi.MirrorConfig, cacheRegistryConfig *cacheapi.RegistryConfig, fldPath *field.Path) field.ErrorList {
	upstreams := sets.New[string]()
	for _, cache := range cacheRegistryConfig.Caches {
		upstreams.Insert(cache.Upstream)
	}

	var allErrs field.ErrorList
	for i, mirror := range mirrorConfig.Mirrors {
		configFldPath := fldPath.Child("mirrors").Index(i)

		if upstreams.Has(mirror.Upstream) {
			allErrs = append(allErrs, field.Invalid(configFldPath.Child("upstream"), mirror.Upstream, fmt.Sprintf("upstream host '%s' is also configured as a registry cache upstream", mirror.Upstream)))
		} else {
			upstreams.Insert(mirror.Upstream)
		}
	}

	return allErrs
}
