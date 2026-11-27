// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

// +k8s:deepcopy-gen=package
// +k8s:conversion-gen=github.com/gardener/gardener-extension-registry-cache/pkg/apis/spegel
// +k8s:defaulter-gen=TypeMeta
// +k8s:openapi-gen=true

//go:generate crd-ref-docs --source-path=. --config=../../../../hack/api-reference/spegel.yaml --renderer=markdown --templates-dir=$GARDENER_HACK_DIR/api-reference/template --log-level=ERROR --output-path=../../../../hack/api-reference/spegel.md

// Package v1alpha1 is a version of the API.
// +groupName=spegel.extensions.gardener.cloud
package v1alpha1 // import "github.com/gardener/gardener-extension-registry-cache/pkg/apis/spegel/v1alpha1"
