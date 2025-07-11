// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
)

var (
	schemeBuilder = runtime.NewSchemeBuilder(
		v1alpha3.AddToScheme,
		registry.AddToScheme,
		setVersionPriority,
	)

	// AddToScheme adds all APIs to the scheme.
	AddToScheme = schemeBuilder.AddToScheme
)

func setVersionPriority(scheme *runtime.Scheme) error {
	return scheme.SetVersionPriority(v1alpha3.SchemeGroupVersion)
}

// Install installs all APIs in the scheme.
func Install(scheme *runtime.Scheme) {
	utilruntime.Must(AddToScheme(scheme))
}
