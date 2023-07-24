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

package app

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/heartbeat"
	"github.com/gardener/gardener/extensions/pkg/util"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	componentbaseconfig "k8s.io/component-base/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	registryinstall "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/install"
	"github.com/gardener/gardener-extension-registry-cache/pkg/controller"
	"github.com/gardener/gardener-extension-registry-cache/pkg/controller/healthcheck"
)

// NewServiceControllerCommand creates a new command that is used to start the registry service controller.
func NewServiceControllerCommand() *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:           "registry-cache",
		Short:         "Registry cache controller manages registry caches within a shoot.",
		SilenceErrors: true,

		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.optionAggregator.Complete(); err != nil {
				return fmt.Errorf("error completing options: %w", err)
			}

			if err := options.heartbeatOptions.Validate(); err != nil {
				return err
			}
			cmd.SilenceUsage = true
			return options.run(cmd.Context())
		},
	}

	options.optionAggregator.AddFlags(cmd.Flags())

	return cmd
}

func (o *Options) run(ctx context.Context) error {
	// TODO: Make these flags configurable via command line parameters or component config file.
	util.ApplyClientConnectionConfigurationToRESTConfig(&componentbaseconfig.ClientConnectionConfiguration{
		QPS:   100.0,
		Burst: 130,
	}, o.restOptions.Completed().Config)

	mgrOpts := o.managerOptions.Completed().Options()

	mgrOpts.ClientDisableCacheFor = []client.Object{
		&corev1.Secret{},    // applied for ManagedResources
		&corev1.ConfigMap{}, // applied for monitoring config
	}

	mgr, err := manager.New(o.restOptions.Completed().Config, mgrOpts)
	if err != nil {
		return fmt.Errorf("could not instantiate controller-manager: %w", err)
	}

	if err := extensionscontroller.AddToScheme(mgr.GetScheme()); err != nil {
		return fmt.Errorf("could not update manager scheme: %w", err)
	}

	if err := registryinstall.AddToScheme(mgr.GetScheme()); err != nil {
		return fmt.Errorf("could not update manager scheme: %w", err)
	}

	ctrlConfig := o.registryOptions.Completed()
	ctrlConfig.ApplyHealthCheckConfig(&healthcheck.DefaultAddOptions.HealthCheckConfig)
	ctrlConfig.Apply(&controller.DefaultAddOptions.Config)
	o.controllerOptions.Completed().Apply(&controller.DefaultAddOptions.ControllerOptions)
	o.healthOptions.Completed().Apply(&healthcheck.DefaultAddOptions.Controller)
	o.reconcileOptions.Completed().Apply(&controller.DefaultAddOptions.IgnoreOperationAnnotation)
	o.heartbeatOptions.Completed().Apply(&heartbeat.DefaultAddOptions)

	if err := o.controllerSwitches.Completed().AddToManager(ctx, mgr); err != nil {
		return fmt.Errorf("could not add controllers to manager: %w", err)
	}

	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("error running manager: %w", err)
	}

	return nil
}
