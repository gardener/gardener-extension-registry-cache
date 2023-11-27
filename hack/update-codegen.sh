#!/usr/bin/env bash
#
# Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
  deepcopy,defaulter \
  github.com/gardener/gardener-extension-registry-cache/pkg/client \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  "registry:v1alpha1,v1alpha2" \
  --go-header-file "${PROJECT_ROOT}/vendor/github.com/gardener/gardener/hack/LICENSE_BOILERPLATE.txt"

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  conversion \
  github.com/gardener/gardener-extension-registry-cache/pkg/client \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  "registry:v1alpha1,v1alpha2" \
  --go-header-file "${PROJECT_ROOT}/vendor/github.com/gardener/gardener/hack/LICENSE_BOILERPLATE.txt"

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter \
  github.com/gardener/gardener-extension-registry-cache/pkg/client \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  "config:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/vendor/github.com/gardener/gardener/hack/LICENSE_BOILERPLATE.txt"

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  conversion \
  github.com/gardener/gardener-extension-registry-cache/pkg/client \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  github.com/gardener/gardener-extension-registry-cache/pkg/apis \
  "config:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/vendor/github.com/gardener/gardener/hack/LICENSE_BOILERPLATE.txt"
