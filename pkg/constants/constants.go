// Copyright (c) 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
