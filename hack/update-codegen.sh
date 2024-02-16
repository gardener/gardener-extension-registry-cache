#!/usr/bin/env bash

# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

# Friendly reminder if workspace location is not in $GOPATH
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
if [ "${SCRIPT_DIR}" != "$(realpath $GOPATH)/src/github.com/gardener/gardener-extension-registry-cache/hack" ]; then
  echo "'hack/update-codegen.sh' script does not work correctly if your workspace is outside GOPATH"
  echo "Please check https://github.com/gardener/gardener/blob/master/docs/development/local_setup.md#get-the-sources"
  exit 1
fi

# We need to explicitly pass GO111MODULE=off to k8s.io/code-generator as it is significantly slower otherwise,
# see https://github.com/kubernetes/code-generator/issues/100.
export GO111MODULE=off

rm -f $GOPATH/bin/*-gen

PROJECT_ROOT=$(dirname $0)/..

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter,conversion \
  github.com/gardener/gardener-extension-registry-cache/pkg/client \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  "registry:v1alpha2,v1alpha3" \
  --go-header-file "${PROJECT_ROOT}/hack/LICENSE_BOILERPLATE.txt"

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter,conversion \
  github.com/gardener/gardener-extension-registry-cache/pkg/client \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  "mirror:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/LICENSE_BOILERPLATE.txt"

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter,conversion \
  github.com/gardener/gardener-extension-registry-cache/pkg/client \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  "config:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/LICENSE_BOILERPLATE.txt"
