#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

function with_backoff {
  if [[ -z "$run_id" ]]; then
    echo "run_id is required!"
    exit 1
  fi

  local max_attempts=${MAX_ATTEMPTS-60}
  local linear_duration=${LINEAR_DURATION-150}
  local interval=${INTERVAL-3}
  local max_interval=192
  local attempt=1
  local exit_code=0

  while (( attempt <= max_attempts ))
  do
    if "$@"
    then
      return 0
    else
      exit_code=$?
    fi

    echo "[$run_id] Request failed! Retrying in $interval seconds..."
    sleep "$interval"

    if (( attempt * (interval + 2) >= linear_duration && interval * 2 <= max_interval )); then
      interval=$(( interval * 2 ))
    fi
    attempt=$(( attempt + 1 ))

  done

  if [[ $exit_code != 0 ]]
  then
    echo "[$run_id] Request failed for $max_attempts time!" 1>&2
  fi

  return $exit_code
}

function configure_registry {
  upstream_host=$1
  registry_cache_endpoint=$2
  upstream_url=$3

  if ! run_id=$upstream_host with_backoff curl --silent --show-error --connect-timeout 2 "$registry_cache_endpoint"
  then
    echo "[$upstream_host] Failed why waiting registry to be available. Exiting..."
    exit 1
  fi

  echo "[$upstream_host] Registry is available. Creating hosts.toml file..."

  config_path="/etc/containerd/certs.d"
  mkdir -p "$config_path/$upstream_host"
  cat <<EOF > "$config_path/$upstream_host/hosts.toml"
server = "$upstream_url"

[host."$registry_cache_endpoint"]
  capabilities = ["pull", "resolve"]
EOF

  echo "[$upstream_host] Created hosts.toml file."  
}

pids=()

for arg in "$@"
do
  IFS=',' read -ra parsed_arg <<< "$arg"

  upstream_host=${parsed_arg[0]}
  registry_cache_endpoint=${parsed_arg[1]}
  upstream_url=${parsed_arg[2]}

  echo "[$upstream_host] Configuring registry..."
  configure_registry "$upstream_host" "$registry_cache_endpoint" "$upstream_url" &
  pids+=($!)
done

echo "Waiting for processes [${pids[*]}] to finish..."
wait "${pids[@]}"
