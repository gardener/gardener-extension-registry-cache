// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// +k8s:deepcopy-gen=package
// +k8s:conversion-gen=github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror
// +k8s:defaulter-gen=TypeMeta
// +k8s:openapi-gen=true

//go:generate crd-ref-docs --source-path=. --config=../../../../hack/api-reference/mirror.yaml --renderer=markdown --templates-dir=$GARDENER_HACK_DIR/api-reference/template  --log-level=ERROR --output-path=../../../../hack/api-reference/mirror.md

// Package v1alpha1 is a version of the API.
// +groupName=mirror.extensions.gardener.cloud
package v1alpha1
