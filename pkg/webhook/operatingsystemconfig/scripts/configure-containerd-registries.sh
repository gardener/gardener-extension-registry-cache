#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

# with_retry retries the given command until success or until the retry attempts exceed MAX_TOTAL_ATTEMPTS.
# First, MAX_LINEAR_ATTEMPTS amount of retries are performed on every INTERVAL seconds.
# After, the rest of the retry attempts are executed with exponential backoff with base INTERVAL and factor 2.
function with_retry {
  if [[ -z "$run_id" ]]; then
    echo "run_id is required!"
    exit 1
  fi

  local -r max_total_attempts=${MAX_TOTAL_ATTEMPTS-50}
  local -r max_linear_attempts=${MAX_LINEAR_ATTEMPTS-30}
  local -r interval=${INTERVAL-3}

  if [[ $max_total_attempts -lt $max_linear_attempts ]]; then
    echo "MAX_TOTAL_ATTEMPTS must be not be less than MAX_LINEAR_ATTEMPTS"
    exit 1
  fi

  local delay=$interval
  local attempt=1
  local exit_code=0

  while (( attempt <= max_total_attempts ))
  do
    if "$@"
    then
      return 0
    else
      exit_code=$?
    fi

    echo "[$run_id] Request failed! Retrying in $delay seconds..."
    sleep "$delay"

    attempt=$(( attempt + 1 ))
    if (( attempt > max_linear_attempts )); then
      exponential_attempt=$(( attempt - max_linear_attempts ))
      delay=$(( ( exponential_attempt ** 2 ) * interval ))
    fi
  done

  if [[ $exit_code != 0 ]]
  then
    echo "[$run_id] Request failed for $max_total_attempts time!" 1>&2
  fi

  return $exit_code
}

function configure_registry {
  upstream_host=$1
  registry_cache_endpoint=$2
  upstream_url=$3

  if ! run_id=$upstream_host with_retry curl --silent --show-error --connect-timeout 2 "$registry_cache_endpoint"
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
