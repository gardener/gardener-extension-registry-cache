---
title: Deploying Registry Cache Extension in Gardener's Local Setup with Provider Extensions
description: Learn how to set up a development environment using own Seed clusters on an existing Kubernetes cluster
---

# Deploying Registry Cache Extension in Gardener's Local Setup with Provider Extensions

## Prerequisites

- Make sure that you have a running local Gardener setup with enabled provider extensions. The steps to complete this can be found in the [Deploying Gardener Locally and Enabling Provider-Extensions](https://github.com/gardener/gardener/blob/master/docs/deployment/getting_started_locally_with_extensions.md) guide.

> [!TIP]
> Ensure that the locally used Gardener version matches the version specified by the `github.com/gardener/gardener` dependency.
> The extension’s local setup must run successfully against a local Gardener setup at the version referenced by this dependency.

## Setting up the Registry Cache Extension

Make sure that your `KUBECONFIG` environment variable is targeting the local Gardener cluster.

The location of the Gardener project from the Gardener setup step is expected to be under the same root (e.g. `~/go/src/github.com/gardener/`). If this is not the case, the location of Gardener project should be specified in `GARDENER_REPO_ROOT` environment variable:

```bash
export GARDENER_REPO_ROOT="<path_to_gardener_project>"
```

Then you can run:

```bash
make remote-extension-up
```

In case you have added additional Seeds you can specify the seed name:

```bash
make remote-extension-up SEED_NAME=<seed-name>
```

The corresponding make target will build the extension image, push it into the Seed cluster image registry, and deploy the registry-cache ControllerDeployment and ControllerRegistration resources into the kind cluster.
The container image in the ControllerDeployment will be the image that was build and pushed into the Seed cluster image registry.

The make target will then deploy the registry-cache admission component. It will build the admission image, push it into the kind cluster image registry, and finally install the admission component charts to the kind cluster.

## Creating a Shoot Cluster

Once the above step is completed, you can create a Shoot cluster. In order to create a Shoot cluster, please create your own Shoot definition depending on providers on your Seed cluster.

## Tearing Down the Development Environment

To tear down the development environment, delete the Shoot cluster or disable the `registry-cache` extension in the Shoot's specification. When the extension is not used by the Shoot anymore, you can run:

```bash
make remote-extension-down
```

The make target will delete the ControllerDeployment and ControllerRegistration of the extension, and the registry-cache admission helm deployment.
