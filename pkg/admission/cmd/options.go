// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	webhookcmd "github.com/gardener/gardener/extensions/pkg/webhook/cmd"

	cachevalidator "github.com/gardener/gardener-extension-registry-cache/pkg/admission/validator/cache"
	mirrorvalidator "github.com/gardener/gardener-extension-registry-cache/pkg/admission/validator/mirror"
)

// GardenWebhookSwitchOptions are the webhookcmd.SwitchOptions for the admission webhooks.
func GardenWebhookSwitchOptions() *webhookcmd.SwitchOptions {
	return webhookcmd.NewSwitchOptions(
		webhookcmd.Switch(cachevalidator.Name, cachevalidator.New),
		webhookcmd.Switch(mirrorvalidator.Name, mirrorvalidator.New),
	)
}
