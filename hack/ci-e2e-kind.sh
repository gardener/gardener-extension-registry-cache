#!/usr/bin/env bash

# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o nounset
set -o pipefail
set -o errexit

clamp_mss_to_pmtu() {
  # https://github.com/kubernetes/test-infra/issues/23741
  if [[ "$OSTYPE" != "darwin"* ]]; then
    iptables -t mangle -A POSTROUTING -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu
  fi
}

ensure_local_gardener_cloud_hosts() {
  if [ -n "${CI:-}" -a -n "${ARTIFACTS:-}" ]; then
    echo "> Adding local.gardener.cloud entries to /etc/hosts..."
    printf "\n127.0.0.1 registry.local.gardener.cloud\n" >> /etc/hosts
    printf "\n::1 registry.local.gardener.cloud\n" >> /etc/hosts
    echo "> Content of '/etc/hosts' after adding local.gardener.cloud entries:\n$(cat /etc/hosts)"
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

ensure_local_gardener_cloud_hosts

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
