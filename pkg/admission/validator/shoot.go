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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/validation"
	"github.com/gardener/gardener-extension-registry-cache/pkg/controller"
)

// shoot validates shoots
type shoot struct {
	client  client.Client
	decoder runtime.Decoder
}

// NewShootValidator returns a new instance of a shoot validator.
func NewShootValidator(client client.Client, decoder runtime.Decoder) extensionswebhook.Validator {
	return &shoot{
		client:  client,
		decoder: decoder,
	}
}

// Validate validates the given shoot object
func (s *shoot) Validate(_ context.Context, new, _ client.Object) error {
	shoot, ok := new.(*core.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	var ext *core.Extension
	var fldPath *field.Path
	for i, ex := range shoot.Spec.Extensions {
		if ex.Type == controller.Type {
			ext = ex.DeepCopy()
			fldPath = field.NewPath("spec", "extensions").Index(i)
			break
		}
	}
	if ext == nil {
		return nil
	}

	for _, worker := range shoot.Spec.Provider.Workers {
		if worker.CRI.Name != "containerd" {
			return fmt.Errorf("containerruntime needs to be containerd when container registry cache is used")
		}
	}

	providerConfigPath := fldPath.Child("providerConfig")

	registryConfig, err := decodeRegistryConfig(s.decoder, ext.ProviderConfig, providerConfigPath)
	if err != nil {
		return err
	}

	return validation.ValidateRegistryConfig(registryConfig, providerConfigPath).ToAggregate()

}
