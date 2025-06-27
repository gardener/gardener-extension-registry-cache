// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package mirror

import (
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/component/extensions/operatingsystemconfig/original/components/kubelet"
	oscutils "github.com/gardener/gardener/pkg/component/extensions/operatingsystemconfig/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	// Name is the webhook name.
	Name = "registry-mirror"
)

var logger = log.Log.WithName("registry-mirror-webhook")

// New returns a new mutating webhook that for a registry-mirror Extension adds the required containerd registry configuration files to the OperatingSystemConfig resource.
func New(mgr manager.Manager) (*extensionswebhook.Webhook, error) {
	logger.Info("Adding webhook to manager")

	fciCodec := oscutils.NewFileContentInlineCodec()

	decoder := serializer.NewCodecFactory(mgr.GetScheme()).UniversalDecoder()

	mutator := genericmutator.NewMutator(
		mgr,
		NewEnsurer(mgr.GetClient(), decoder, logger),
		oscutils.NewUnitSerializer(),
		kubelet.NewConfigCodec(fciCodec),
		fciCodec,
		logger,
	)
	types := []extensionswebhook.Type{
		{Obj: &extensionsv1alpha1.OperatingSystemConfig{}},
	}
	handler, err := extensionswebhook.NewBuilder(mgr, logger).WithMutator(mutator, types...).Build()
	if err != nil {
		return nil, err
	}

	webhook := &extensionswebhook.Webhook{
		Name:     Name,
		Provider: "",
		Types:    types,
		Target:   extensionswebhook.TargetSeed,
		Path:     "/webhooks/registry-mirror",
		Webhook:  &admission.Webhook{Handler: handler},
		NamespaceSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{v1beta1constants.LabelExtensionPrefix + "registry-mirror": "true"},
		},
	}

	return webhook, nil
}
