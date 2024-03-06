---
title: Deploying Registry Cache Extension Locally
description: Learn how to set up a local development environment
---

# Deploying Registry Cache Extension Locally

## Prerequisites

- Make sure that you have a running local Gardener setup. The steps to complete this can be found in the [Deploying Gardener Locally guide](https://github.com/gardener/gardener/blob/master/docs/deployment/getting_started_locally.md).

## Setting up the Registry Cache Extension

Make sure that your `KUBECONFIG` environment variable is targeting the local Gardener cluster. When this is ensured, run:

```bash
make extension-up
```

The corresponding `make` target will build the extension image, load it into the kind cluster Nodes, and deploy the registry-cache ControllerDeployment and ControllerRegistration resources. The container image in the ControllerDeployment will be the image that was build and loaded into the kind cluster Nodes.

The `make` target will then deploy the registry-cache admission component. It will build the admission image, load it into the kind cluster Nodes, and finally install the admission component charts to the kind cluster.

## Creating a Shoot Cluster

Once the above step is completed, you can create a Shoot cluster.

[`example/shoot-registry-cache.yaml`](../../example/shoot-registry-cache.yaml) contains a Shoot specification with the `registry-cache` extension:
```bash
kubectl create -f example/shoot-registry-cache.yaml
```

[`example/shoot-registry-mirror.yaml`](../../example/shoot-registry-mirror.yaml) contains a Shoot specification with the `registry-mirror` extension:
```bash
kubectl create -f example/shoot-registry-mirror.yaml
```

## Tearing Down the Development Environment

To tear down the development environment, delete the Shoot cluster or disable the `registry-cache` extension in the Shoot's specification. When the extension is not used by the Shoot anymore, you can run:

```bash
make extension-down
```

The `make` target will delete the ControllerDeployment and ControllerRegistration of the extension, and the registry-cache admission helm deployment.
