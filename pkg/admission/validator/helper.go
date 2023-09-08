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

package validator

import (
	"github.com/gardener/gardener/pkg/apis/core"

	"github.com/gardener/gardener-extension-registry-cache/pkg/constants"
)

// FindRegistryCacheExtension finds the registry-cache extension.
// The first return argument is whether the extension was found.
// The second return argument is index of the extension in the list. -1 is returned if the extension is not found.
// The third return arguement is the extension itself. An empty extension is returned if the extension is not found.
func FindRegistryCacheExtension(extensions []core.Extension) (bool, int, core.Extension) {
	for i, ext := range extensions {
		if ext.Type == constants.ExtensionType {
			return true, i, ext
		}
	}

	return false, -1, core.Extension{}
}
