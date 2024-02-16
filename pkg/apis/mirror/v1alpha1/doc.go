// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// +k8s:deepcopy-gen=package
// +k8s:conversion-gen=github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror
// +k8s:defaulter-gen=TypeMeta
// +k8s:openapi-gen=true

//go:generate gen-crd-api-reference-docs -api-dir . -config ../../../../hack/api-reference/config.json -template-dir "$GARDENER_HACK_DIR/api-reference/template" -out-file ../../../../hack/api-reference/mirror.md

// Package v1alpha1 is a version of the API.
// +groupName=mirror.extensions.gardener.cloud
package v1alpha1
