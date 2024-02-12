// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/config"
)

// Config contains configuration for the registry cache service.
type Config struct {
	config.Configuration
}
