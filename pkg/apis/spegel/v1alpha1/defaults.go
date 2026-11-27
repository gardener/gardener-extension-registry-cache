// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

// SetDefaults_SpegelConfig sets the defaults for the Spegel ports and the resolve tags setting.
func SetDefaults_SpegelConfig(spegelConfig *SpegelConfig) {
	if spegelConfig.RegistryPort == nil {
		spegelConfig.RegistryPort = new(int32(15500))
	}
	if spegelConfig.RouterPort == nil {
		spegelConfig.RouterPort = new(int32(15501))
	}
	if spegelConfig.MetricsPort == nil {
		spegelConfig.MetricsPort = new(int32(19090))
	}
	if spegelConfig.ResolveTags == nil {
		spegelConfig.ResolveTags = new(bool(true))
	}
}
