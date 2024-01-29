// Copyright (c) 2024 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package mirror

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MirrorConfig contains information about registry mirrors to configure.
type MirrorConfig struct {
	metav1.TypeMeta

	// Mirrors is a slice of registry mirrors to configure.
	Mirrors []MirrorConfiguration
}

// MirrorConfiguration represents a registry mirror.
type MirrorConfiguration struct {
	// Upstream is the remote registry host to mirror.
	Upstream string
	// Hosts are the mirror hosts to be used for the upstream.
	Hosts []MirrorHost
}

// MirrorHost represents a mirror host.
type MirrorHost struct {
	// Host is the mirror host.
	Host string
}
