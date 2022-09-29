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

package controller

import (
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/config"
)

const (
	// Type is the type of Extension resource.
	Type = "registry-cache"
	// ControllerName is the name of the registry cache service controller.
	ControllerName = "registry-cache"
	// FinalizerSuffix is the finalizer suffix for the registry cache service controller.
	FinalizerSuffix = "registry-cache"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{}
)

// AddOptions are options to apply when adding the registry cache service controller to the manager.
type AddOptions struct {
	// ControllerOptions contains options for the controller.
	ControllerOptions controller.Options
	// Config contains configuration for the registry cache service.
	Config config.Configuration
	// IgnoreOperationAnnotation specifies whether to ignore the operation annotation or not.
	IgnoreOperationAnnotation bool
}

// AddToManager adds a controller with the default Options to the given Controller Manager.
func AddToManager(mgr manager.Manager) error {
	return AddToManagerWithOptions(mgr, DefaultAddOptions)
}

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(mgr manager.Manager, opts AddOptions) error {
	return extension.Add(mgr, extension.AddArgs{
		Actuator:          NewActuator(opts.Config),
		ControllerOptions: opts.ControllerOptions,
		Name:              ControllerName,
		FinalizerSuffix:   FinalizerSuffix,
		Resync:            0,
		Predicates:        extension.DefaultPredicates(DefaultAddOptions.IgnoreOperationAnnotation),
		Type:              Type,
	})
}
