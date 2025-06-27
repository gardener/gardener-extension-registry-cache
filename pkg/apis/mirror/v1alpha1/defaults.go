// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

// SetDefaults_MirrorHost sets the defaults for a MirrorHost.
func SetDefaults_MirrorHost(mirrorHost *MirrorHost) {
	if len(mirrorHost.Capabilities) == 0 {
		mirrorHost.Capabilities = []MirrorHostCapability{MirrorHostCapabilityPull}
	}
}
