// Copyright 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package webhook

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Validator validates objects.
type Validator interface {
	Validate(ctx context.Context, new, old client.Object) error
}

type validationWrapper struct {
	Validator
}

// Mutate implements the `Mutator` interface and calls the `Validate` function of the underlying validator.
func (d *validationWrapper) Mutate(ctx context.Context, new, old client.Object) error {
	return d.Validate(ctx, new, old)
}

func hybridValidator(val Validator) Mutator {
	return &validationWrapper{val}
}
