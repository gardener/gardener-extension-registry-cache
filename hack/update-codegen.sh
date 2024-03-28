#!/usr/bin/env bash

# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

# setup virtual GOPATH
source "$GARDENER_HACK_DIR"/vgopath-setup.sh

CODE_GEN_DIR=$(go list -m -f '{{.Dir}}' k8s.io/code-generator)

# We need to explicitly pass GO111MODULE=off to k8s.io/code-generator as it is significantly slower otherwise,
# see https://github.com/kubernetes/code-generator/issues/100.
export GO111MODULE=off

rm -f $GOPATH/bin/*-gen

PROJECT_ROOT=$(dirname $0)/..

bash "${CODE_GEN_DIR}/generate-internal-groups.sh" \
  deepcopy,defaulter,conversion \
  github.com/gardener/gardener-extension-registry-cache/pkg/client \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  "registry:v1alpha3" \
  --go-header-file "${PROJECT_ROOT}/hack/LICENSE_BOILERPLATE.txt"

bash "${CODE_GEN_DIR}/generate-internal-groups.sh" \
  deepcopy,defaulter,conversion \
  github.com/gardener/gardener-extension-registry-cache/pkg/client \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  "mirror:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/LICENSE_BOILERPLATE.txt"

bash "${CODE_GEN_DIR}/generate-internal-groups.sh" \
  deepcopy,defaulter,conversion \
  github.com/gardener/gardener-extension-registry-cache/pkg/client \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  "config:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/LICENSE_BOILERPLATE.txt"
