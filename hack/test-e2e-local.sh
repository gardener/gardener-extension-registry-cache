#!/usr/bin/env bash

# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

echo "> E2E Tests"

local_address="172.18.255.1"
# We have to make the shoot domains accessible.
seed_name="local"

shoot_names=(
  e2e-cache-def.local
  e2e-cache-hib.local
  e2e-cache-ssc.local
  e2e-cache-fd.local
  e2e-cache-pr.local
  e2e-mirror-def.local
)

if [ -n "${CI:-}" ]; then
  for shoot in "${shoot_names[@]}" ; do
    printf "\n$local_address api.%s.external.local.gardener.cloud\n$local_address api.%s.internal.local.gardener.cloud\n" $shoot $shoot >>/etc/hosts
  done
else
  missing_entries=()

  for shoot in "${shoot_names[@]}"; do
    for ip in internal external; do
      if ! grep -q -x "$local_address api.$shoot.$ip.local.gardener.cloud" /etc/hosts; then
        missing_entries+=("$local_address api.$shoot.$ip.local.gardener.cloud")
      fi
    done
  done

  if [ ${#missing_entries[@]} -gt 0 ]; then
    printf "Hostnames for the following Shoots are missing in /etc/hosts:\n"
    for entry in "${missing_entries[@]}"; do
      printf " - %s\n" "$entry"
    done
    printf "To access shoot clusters and run e2e tests, you have to extend your /etc/hosts file.\nPlease refer to https://github.com/gardener/gardener/blob/master/docs/deployment/getting_started_locally.md#accessing-the-shoot-cluster\n\n"
    exit 1
  fi
fi

GO111MODULE=on ginkgo run --timeout=1h30m --v --show-node-events "$@"
