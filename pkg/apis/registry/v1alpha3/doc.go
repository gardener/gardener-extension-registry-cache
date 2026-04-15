// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// +k8s:deepcopy-gen=package
// +k8s:conversion-gen=github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry
// +k8s:defaulter-gen=TypeMeta
// +k8s:openapi-gen=true

//go:generate crd-ref-docs --source-path=. --config=../../../../hack/api-reference/registry.yaml --renderer=markdown --templates-dir=$GARDENER_HACK_DIR/api-reference/template --log-level=ERROR --output-path=../../../../hack/api-reference/registry.md

// Package v1alpha3 is a version of the API.
// +groupName=registry.extensions.gardener.cloud
package v1alpha3 // import "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
