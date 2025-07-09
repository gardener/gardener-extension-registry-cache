// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	extensionscmdwebhook "github.com/gardener/gardener/extensions/pkg/webhook/cmd"

	cachevalidator "github.com/gardener/gardener-extension-registry-cache/pkg/admission/validator/cache"
	mirrorvalidator "github.com/gardener/gardener-extension-registry-cache/pkg/admission/validator/mirror"
)

// GardenWebhookSwitchOptions are the extensionscmdwebhook.SwitchOptions for the admission webhooks.
func GardenWebhookSwitchOptions() *extensionscmdwebhook.SwitchOptions {
	return extensionscmdwebhook.NewSwitchOptions(
		extensionscmdwebhook.Switch(cachevalidator.Name, cachevalidator.New),
		extensionscmdwebhook.Switch(mirrorvalidator.Name, mirrorvalidator.New),
	)
}
