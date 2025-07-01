# [Gardener Extension for Registry Cache](https://gardener.cloud)

[![REUSE status](https://api.reuse.software/badge/github.com/gardener/gardener-extension-registry-cache)](https://api.reuse.software/info/github.com/gardener/gardener-extension-registry-cache)
[![CI Build status](https://github.com/gardener/gardener-extension-registry-cache/actions/workflows/non-release.yaml/badge.svg)](https://github.com/gardener/gardener-extension-registry-cache/actions/workflows/non-release.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/gardener/gardener-extension-registry-cache)](https://goreportcard.com/report/github.com/gardener/gardener-extension-registry-cache)

Gardener extension controller which deploys pull-through caches for container registries.

## Usage

- [Configuring the Registry Cache Extension](docs/usage/registry-cache/configuration.md) - learn what is the use-case for a pull-through cache, how to enable it and configure it
- [How to provide credentials for upstream repository?](docs/usage/registry-cache/upstream-credentials.md)
- [Registry Cache Observability](docs/usage/registry-cache/observability.md) - learn what metrics and alerts are exposed and how to view the registry cache logs
- [Configuring the Registry Mirror Extension](docs/usage/registry-mirror/configuration.md) - learn what is the use-case for a registry mirror, how to enable and configure it

## Local Setup and Development

- [Deploying Registry Cache Extension Locally](docs/development/getting-started-locally.md) - learn how to set up a local development environment
- [Deploying Registry Cache Extension in Gardener's Local Setup with Provider Extensions](docs/development/getting-started-remotely.md) - learn how to set up a development environment using own Seed clusters on an existing Kubernetes cluster
- [Developer Docs for Gardener Extension Registry Cache](docs/development/extension-registry-cache.md) - learn about the inner workings
