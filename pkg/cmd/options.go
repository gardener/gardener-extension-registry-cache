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

package cmd

import (
	"errors"
	"os"

	extensionsapisconfig "github.com/gardener/gardener/extensions/pkg/apis/config"
	"github.com/gardener/gardener/extensions/pkg/controller/cmd"
	extensionshealthcheckcontroller "github.com/gardener/gardener/extensions/pkg/controller/healthcheck"
	extensionsheartbeatcontroller "github.com/gardener/gardener/extensions/pkg/controller/heartbeat"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	configapi "github.com/gardener/gardener-extension-registry-cache/pkg/apis/config"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/config/v1alpha1"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/config/validation"
	"github.com/gardener/gardener-extension-registry-cache/pkg/controller"
	healthcheckcontroller "github.com/gardener/gardener-extension-registry-cache/pkg/controller/healthcheck"
)

var (
	scheme  *runtime.Scheme
	decoder runtime.Decoder
)

func init() {
	scheme = runtime.NewScheme()
	utilruntime.Must(configapi.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
}

// RegistryOptions holds options related to the registry service.
type RegistryOptions struct {
	ConfigLocation string
	config         *RegistryServiceConfig
}

// AddFlags implements Flagger.AddFlags.
func (o *RegistryOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ConfigLocation, "config", "", "Path to registry service configuration")
}

// Complete implements Completer.Complete.
func (o *RegistryOptions) Complete() error {
	if o.ConfigLocation == "" {
		return errors.New("config location is not set")
	}
	data, err := os.ReadFile(o.ConfigLocation)
	if err != nil {
		return err
	}

	config := configapi.Configuration{}
	_, _, err = decoder.Decode(data, nil, &config)
	if err != nil {
		return err
	}

	if errs := validation.ValidateConfiguration(&config); len(errs) > 0 {
		return errs.ToAggregate()
	}

	o.config = &RegistryServiceConfig{
		config: config,
	}

	return nil
}

// Completed returns the decoded RegistryServiceConfiguration instance. Only call this if `Complete` was successful.
func (o *RegistryOptions) Completed() *RegistryServiceConfig {
	return o.config
}

// RegistryServiceConfig contains configuration information about the registry service.
type RegistryServiceConfig struct {
	config configapi.Configuration
}

// Apply applies the RegistryOptions to the passed ControllerOptions instance.
func (c *RegistryServiceConfig) Apply(config *configapi.Configuration) {
	*config = c.config
}

// ControllerSwitches are the cmd.SwitchOptions for the provider controllers.
func ControllerSwitches() *cmd.SwitchOptions {
	return cmd.NewSwitchOptions(
		cmd.Switch(controller.ControllerName, controller.AddToManager),
		cmd.Switch(extensionshealthcheckcontroller.ControllerName, healthcheckcontroller.AddToManager),
		cmd.Switch(extensionsheartbeatcontroller.ControllerName, extensionsheartbeatcontroller.AddToManager),
	)
}

// ApplyHealthCheckConfig applies the HealthCheckConfig.
func (c *RegistryServiceConfig) ApplyHealthCheckConfig(config *extensionsapisconfig.HealthCheckConfig) {
	if c.config.HealthCheckConfig != nil {
		*config = *c.config.HealthCheckConfig
	}
}
