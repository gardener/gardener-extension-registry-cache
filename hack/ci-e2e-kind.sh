#!/usr/bin/env bash
#
# Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

set -o nounset
set -o pipefail
set -o errexit

clamp_mss_to_pmtu() {
  # https://github.com/kubernetes/test-infra/issues/23741
  if [[ "$OSTYPE" != "darwin"* ]]; then
    iptables -t mangle -A POSTROUTING -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu
  fi
}

REPO_ROOT="$(readlink -f $(dirname ${0})/..)"
GARDENER_VERSION=$(go list -m -f '{{.Version}}' github.com/gardener/gardener)

if [[ ! -d "$REPO_ROOT/gardener" ]]; then
  git clone --branch $GARDENER_VERSION https://github.com/gardener/gardener.git
else
  (cd "$REPO_ROOT/gardener" && git checkout $GARDENER_VERSION)
fi

clamp_mss_to_pmtu

make -C "$REPO_ROOT/gardener" kind-up
export KUBECONFIG=$REPO_ROOT/gardener/example/gardener-local/kind/local/kubeconfig

trap '{
  make -C "$REPO_ROOT/gardener" kind-down
}' EXIT

make -C "$REPO_ROOT/gardener" gardener-up
make extension-up
make test-e2e-local
make extension-down
make -C "$REPO_ROOT/gardener" gardener-down