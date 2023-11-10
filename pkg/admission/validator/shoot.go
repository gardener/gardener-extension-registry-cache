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

package validator

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1/helper"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/validation"
)

// shoot validates shoots
type shoot struct {
	apiReader client.Reader
	decoder   runtime.Decoder
}

// NewShootValidator returns a new instance of a shoot validator.
func NewShootValidator(apiReader client.Reader, decoder runtime.Decoder) extensionswebhook.Validator {
	return &shoot{
		apiReader: apiReader,
		decoder:   decoder,
	}
}

// Validate validates the given shoot object
func (s *shoot) Validate(ctx context.Context, new, old client.Object) error {
	shoot, ok := new.(*core.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	i, ext := FindRegistryCacheExtension(shoot.Spec.Extensions)
	if i == -1 {
		return nil
	}

	for _, worker := range shoot.Spec.Provider.Workers {
		if worker.CRI.Name != "containerd" {
			return fmt.Errorf("container runtime needs to be containerd when the registry-cache extension is enabled")
		}
	}

	providerConfigPath := field.NewPath("spec", "extensions").Index(i).Child("providerConfig")
	if ext.ProviderConfig == nil {
		return field.Required(providerConfigPath, "providerConfig is required for the registry-cache extension")
	}

	registryConfig := &api.RegistryConfig{}
	if err := runtime.DecodeInto(s.decoder, ext.ProviderConfig.Raw, registryConfig); err != nil {
		return fmt.Errorf("failed to decode providerConfig: %w", err)
	}

	allErrs := field.ErrorList{}

	if old != nil {
		oldShoot, ok := old.(*core.Shoot)
		if !ok {
			return fmt.Errorf("wrong object type %T for old object", old)
		}

		oldI, oldExt := FindRegistryCacheExtension(oldShoot.Spec.Extensions)
		if oldI != -1 {
			if oldExt.ProviderConfig == nil {
				return fmt.Errorf("providerConfig is not available on old Shoot")
			}

			oldRegistryConfig := &api.RegistryConfig{}
			if err := runtime.DecodeInto(s.decoder, oldExt.ProviderConfig.Raw, oldRegistryConfig); err != nil {
				return fmt.Errorf("failed to decode providerConfig: %w", err)
			}

			allErrs = append(allErrs, validation.ValidateRegistryConfigUpdate(oldRegistryConfig, registryConfig, providerConfigPath)...)
		}
	}

	errList, err := s.validateRegistryConfig(ctx, registryConfig, providerConfigPath, shoot.Spec.Resources, shoot.Namespace)
	if err != nil {
		return err
	}
	allErrs = append(allErrs, errList...)

	return allErrs.ToAggregate()
}

// validateRegistryConfig validates the passed configuration instance.
func (s *shoot) validateRegistryConfig(ctx context.Context, config *api.RegistryConfig, fldPath *field.Path, resources []core.NamedResourceReference, namespace string) (field.ErrorList, error) {
	allErrs := field.ErrorList{}

	if len(config.Caches) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("caches"), "at least one cache must be provided"))
	}

	upstreams := sets.New[string]()
	for i, cache := range config.Caches {
		cacheFldPath := fldPath.Child("caches").Index(i)
		allErrs = append(allErrs, validation.ValidateRegistryCache(cache, cacheFldPath)...)

		if upstreams.Has(cache.Upstream) {
			allErrs = append(allErrs, field.Duplicate(cacheFldPath.Child("upstream"), cache.Upstream))
		} else {
			upstreams.Insert(cache.Upstream)
		}

		if cache.SecretReferenceName != nil {
			ref := helper.GetSecretReference(resources, *cache.SecretReferenceName)
			secretRefFldPath := cacheFldPath.Child("secretReferenceName")
			if ref == nil {
				allErrs = append(allErrs, field.Invalid(secretRefFldPath, *cache.SecretReferenceName, fmt.Sprintf("referenced resource with kind Secret not found for reference: %q", *cache.SecretReferenceName)))
			} else {
				var (
					secret    = &corev1.Secret{}
					secretKey = kutil.Key(namespace, ref.Name)
				)
				// Explicitly use the client.Reader to prevent controller-runtime to start Informer for Secrets
				// under the hood. The latter increases the memory usage of the component.
				if err := s.apiReader.Get(ctx, secretKey, secret); err != nil {
					return allErrs, fmt.Errorf("failed to get secret %s/%s for secretReferenceName %s: %w", namespace, ref.Name, *cache.SecretReferenceName, err)
				}
				allErrs = append(allErrs, validation.ValidateUpstreamRepositorySecret(secret, secretRefFldPath, *cache.SecretReferenceName)...)
			}
		}
	}

	return allErrs, nil
}
