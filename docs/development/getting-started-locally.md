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

The corresponding make target will build the extension image, load it into the kind cluster Nodes, and deploy the registry-cache ControllerDeployment and ControllerRegistration resources. The container image in the ControllerDeployment will be the image that was build and loaded into the kind cluster Nodes.

The make target will then deploy the registry-cache admission component. It will build the admission image, load it into the kind cluster Nodes, and finally install the admission component charts to the kind cluster.

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

The make target will delete the ControllerDeployment and ControllerRegistration of the extension, and the registry-cache admission helm deployment.

## Alternative Setup Using the `gardener-operator` Local Setup

Alternatively, you can deploy the registry-cache extension in the `gardener-operator` local setup. To do this, make sure you are have a running local setup based on [Alternative Way to Set Up Garden and Seed Leveraging `gardener-operator`](https://github.com/gardener/gardener/blob/master/docs/deployment/getting_started_locally.md#alternative-way-to-set-up-garden-and-seed-leveraging-gardener-operator). The `KUBECONFIG` environment variable should target the operator local KinD cluster (i.e. `<path_to_gardener_project>/example/gardener-local/kind/multi-zone/kubeconfig`).

#### Creating the registry-cache `Extension.operator.gardener.cloud` resource:

```bash
make extension-operator-up
```

The corresponding make target will build the registry-cache admission and extension container images, OCI artifacts for the admission runtime and application charts, and the extension chart. Then, the container images and the OCI artifacts are pushed into the default skaffold registry (i.e. `garden.local.gardener.cloud:5001`). Next, the registry-cache `Extension.operator.gardener.cloud` resource is deployed into the KinD cluster. Based on this resource the gardener-operator will deploy the registry-cache admission component, as well as the registry-cache ControllerDeployment and ControllerRegistration resources.

#### Creating a Shoot Cluster

To create a Shoot cluster the `KUBECONFIG` environment variable should target virtual garden cluster (i.e. `<path_to_gardener_project>/dev-setup/kubeconfigs/virtual-garden/kubeconfig`) and then execute:
```bash
kubectl create -f example/shoot-registry-cache.yaml
```

#### Delete the registry-cache `Extension.operator.gardener.cloud` resource

Make sure the environment variable `KUBECONFIG` points to the operator's local KinD cluster and then run:
```bash
make extension-operator-down
```

The corresponding make target will delete the `Extension.operator.gardener.cloud` resource. Consequently, the gardener-operator will delete the registry-cache admission component and registry-cache ControllerDeployment and ControllerRegistration resources.
