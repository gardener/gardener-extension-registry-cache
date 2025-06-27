// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"os"

	"github.com/gardener/gardener/extensions/pkg/controller/cmd"
	extensionsheartbeatcontroller "github.com/gardener/gardener/extensions/pkg/controller/heartbeat"
	webhookcmd "github.com/gardener/gardener/extensions/pkg/webhook/cmd"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	configapi "github.com/gardener/gardener-extension-registry-cache/pkg/apis/config"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/config/v1alpha1"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/config/validation"
	cachecontroller "github.com/gardener/gardener-extension-registry-cache/pkg/controller/cache"
	mirrorcontroller "github.com/gardener/gardener-extension-registry-cache/pkg/controller/mirror"
	cachewebhook "github.com/gardener/gardener-extension-registry-cache/pkg/webhook/cache"
	mirrorwebhook "github.com/gardener/gardener-extension-registry-cache/pkg/webhook/mirror"
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
	if err := runtime.DecodeInto(decoder, data, &config); err != nil {
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
		cmd.Switch(cachecontroller.ControllerName, cachecontroller.AddToManager),
		cmd.Switch(mirrorcontroller.ControllerName, mirrorcontroller.AddToManager),
		cmd.Switch(extensionsheartbeatcontroller.ControllerName, extensionsheartbeatcontroller.AddToManager),
	)
}

// WebhookSwitchOptions are the webhookcmd.SwitchOptions for the registry-cache webhook.
func WebhookSwitchOptions() *webhookcmd.SwitchOptions {
	return webhookcmd.NewSwitchOptions(
		webhookcmd.Switch(cachewebhook.Name, cachewebhook.New),
		webhookcmd.Switch(mirrorwebhook.Name, mirrorwebhook.New),
	)
}
