#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

PATH_SEED_KUBECONFIG=""

parse_flags() {
  while test $# -gt 0; do
    case "$1" in
    --path-seed-kubeconfig)
      shift; PATH_SEED_KUBECONFIG="$1"
      ;;
    esac
    shift
  done
}

parse_flags "$@"

temp_shoot_info=$(mktemp)
cleanup-shoot-info() {
  rm -f "$temp_shoot_info"
}
trap cleanup-shoot-info EXIT

if kubectl get configmaps -n kube-system shoot-info --kubeconfig "$PATH_SEED_KUBECONFIG" -o yaml > "$temp_shoot_info"; then
    echo "Getting registry domain from shoot"
    registry_domain=reg.$(yq -e '.data.domain' "$temp_shoot_info")
else
  echo "Please enter domain name for registry on the seed"
  echo "Registry domain:"
  read -er registry_domain
fi

echo "Deploying registry-cache admission in garden cluster"
SKAFFOLD_DEFAULT_REPO=garden.local.gardener.cloud:5001 SKAFFOLD_PUSH=true skaffold run -m admission -p remote-extensions

echo "Deploying registry-cache extension"
SKAFFOLD_DEFAULT_REPO=$registry_domain \
  SKAFFOLD_CHECK_CLUSTER_NODE_PLATFORMS="false" \
  SKAFFOLD_PLATFORM="linux/amd64" \
  SKAFFOLD_DISABLE_MULTI_PLATFORM_BUILD="false" \
  SKAFFOLD_PUSH=true \
  skaffold run -m extension
