// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package constants

const (
	// RegistryCacheExtensionType is the name of the registry-cache Extension type.
	RegistryCacheExtensionType = "registry-cache"
	// RegistryMirrorExtensionType is the name of the registry-mirror Extension type.
	RegistryMirrorExtensionType = "registry-mirror"
	// Origin is the origin used for the registry cache ManagedResources.
	Origin = "registry-cache"

	// UpstreamHostLabel is a label on registry cache resources (Service, StatefulSet) which denotes the upstream host.
	UpstreamHostLabel = "upstream-host"
	// RegistryCachePort is the port on which the pull through cache serves requests.
	RegistryCachePort = 5000
)
