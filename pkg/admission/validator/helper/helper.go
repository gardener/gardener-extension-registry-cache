// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"github.com/gardener/gardener/pkg/apis/core"
)

// FindExtension finds extension with the given type.
// The first return argument is index of the extension in the list. -1 is returned if the extension is not found.
// The second return argument is the extension itself. An empty extension is returned if the extension is not found.
func FindExtension(extensions []core.Extension, extensionType string) (int, core.Extension) {
	for i, ext := range extensions {
		if ext.Type == extensionType {
			return i, ext
		}
	}

	return -1, core.Extension{}
}
