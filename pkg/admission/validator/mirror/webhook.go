// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
package mirror

import (
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-registry-cache/pkg/constants"
)

const (
	// Name is a name for a validation webhook.
	Name = "registry-mirror-validator"
)

var logger = log.Log.WithName("registry-mirror-validator-webhook")

// New creates a new webhook that validates Shoot and CloudProfile resources.
func New(mgr manager.Manager) (*extensionswebhook.Webhook, error) {
	logger.Info("Setting up webhook", "name", Name)

	decoder := serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder()

	return extensionswebhook.New(mgr, extensionswebhook.Args{
		Provider: constants.RegistryCacheExtensionType,
		Name:     Name,
		Path:     "/webhooks/registry-config",
		Validators: map[extensionswebhook.Validator][]extensionswebhook.Type{
			NewShootValidator(decoder): {{Obj: &core.Shoot{}}},
		},
		Target: extensionswebhook.TargetSeed,
		ObjectSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"extensions.extensions.gardener.cloud/registry-mirror": "true"},
		},
	})
}
