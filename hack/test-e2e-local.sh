#!/usr/bin/env bash
#
# Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

echo "> E2E Tests"

# We have to make the shoot domains accessible.
seed_name="local"

shoot_names=(
  e2e-cache-def.local
  e2e-cache-hib.local
  e2e-cache-ssc.local
  e2e-cache-fd.local
  e2e-mirror-def.local
)

if [ -n "${CI:-}" ]; then
  for shoot in "${shoot_names[@]}" ; do
    printf "\n127.0.0.1 api.%s.external.local.gardener.cloud\n127.0.0.1 api.%s.internal.local.gardener.cloud\n" $shoot $shoot >>/etc/hosts
  done
else
  missing_entries=()

  for shoot in "${shoot_names[@]}"; do
    for ip in internal external; do
      if ! grep -q -x "127.0.0.1 api.$shoot.$ip.local.gardener.cloud" /etc/hosts; then
        missing_entries+=("127.0.0.1 api.$shoot.$ip.local.gardener.cloud")
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

GO111MODULE=on ginkgo run --timeout=1h --v --show-node-events "$@"
