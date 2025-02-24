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

## Alternatively you can deploy registry cache using gardener operator

To do this, make sure you are have a running local setup based on [gardener-operator](https://github.com/gardener/gardener/blob/master/docs/deployment/getting_started_locally.md#alternative-way-to-set-up-garden-and-seed-leveraging-gardener-operator). The `KUBECONFIG` environment variable should target the operator local KinD cluster (i.e. <gardener/gardener project root>/example/gardener-local/kind/operator/kubeconfig).

- Create registry cache `Extension.operator.gardener.cloud` resource:
  ```bash
  make extension-operator-up
  ```
  The `extension-operator-up` make target will build the registry cache admission and extension images, helm chart OCI artefacts for admission runtime, admission application and extension. Then the images and artefacts are upload into default skaffold registry (i.e. garden.local.gardener.cloud:5001) and **extension-registry-cache** `Extension.operator.gardener.cloud` resource is deployed into the KinD cluster. Based on this resource gardener-operator will deploy registry cache admission component, as well as registry cache ControllerDeployment and ControllerRegistration resources.

- Create Shoot cluster.

  To create a Shoot cluster the `KUBECONFIG` environment variable should target virtual garden cluster (i.e. <gardener/gardener project root>/example/operator/virtual-garden/kubeconfig) and run:
  ```bash
  kubectl create -f example/shoot-registry-cache.yaml
  ```

-  Tearing Down the registry cache `Extension.operator.gardener.cloud` resource.

  Make sure the environment variable `KUBECONFIG` points to the operator's local KinD cluster and then run:
  ```bash
  make extension-operator-down
  ```
  The gardener-operator will delete registry cache admission component and registry cache ControllerDeployment and ControllerRegistration resources. Finally, it will delete the **extension-registry-cache** `Extension.operator.gardener.cloud` resource.
