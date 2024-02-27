---
title: Developer Docs for Gardener Extension Registry Cache
description: Learn about the inner workings
---

# Developer Docs for Gardener Extension Registry Cache

This document outlines how shoot reconciliation and deletion works for a shoot with the registry-cache extension enabled.

## Shoot Reconciliation

This section outlines how the reconciliation works for a shoot with the registry-cache extension enabled.

### Extension Enablement / Reconciliation

This section outlines how the extension enablement/reconciliation works, e.g., the extension has been added to the shoot spec.

1. As part of the shoot reconciliation flow, the gardenlet deploys the [Extension](https://github.com/gardener/gardener/blob/master/docs/extensions/extension.md) resource.
1. The registry-cache extension reconciles the Extension resource. [pkg/controller/cache/actuator.go](../../pkg/controller/cache/actuator.go) contains the implementation of the [extension.Actuator](https://github.com/gardener/gardener/blob/v1.88.0/extensions/pkg/controller/extension/actuator.go) interface. The reconciliation of an extension of type `registry-cache` consists of the following steps:
   1. The extension checks if a registry has been removed (by comparing the status and the spec of the extension). If an upstream is being removed, then it deploys the [`registry-cleaner` DaemonSet](../../pkg/component/registryconfigurationcleaner/registry_configuration_cleaner.go) to the shoot cluster to clean up the existing configuration for the upstream that has to be removed.
   1. The registry-cache extension deploys resources to the shoot cluster via ManagedResource. For every configured upstream, it creates a StatefulSet (with PVC), service, and other resources.
   1. It lists all services from the `kube-system` namespace that have the `upstream-host` label. It will return an error (and retry in exponential backoff) until the services count matches the configured registries count.
   1. When there is a service created for each configured upstream registry, the registry-cache extension populates the extension resource status. In the extension status, for each upstream, it maintains an endpoint (in the format `http://<cluster-ip>:5000`) which can be used to access the registry cache from within the shoot cluster. `<cluster-ip>` is the cluster IP of the registry cache service. The cluster IP of a service is assigned by the Kubernetes API server on service creation.
1. As part of the shoot reconciliation flow, the gardenlet deploys the [OperatingSystemConfig](https://github.com/gardener/gardener/blob/master/docs/extensions/operatingsystemconfig.md) resource.
1. The registry-cache extension serves a webhook that mutates the OperatingSystemConfig resource for shoots having the registry-cache extension enabled (the corresponding namespace gets labeled by the gardenlet with `extensions.gardener.cloud/registry-cache=true`). [pkg/webhook/cache/ensurer.go](../../pkg/webhook/cache/ensurer.go) contains an implementation of the [genericmutator.Ensurer](https://github.com/gardener/gardener/blob/v1.88.0/extensions/pkg/webhook/controlplane/genericmutator/mutator.go) interface.
   1. The webhook appends the [configure-containerd-registries.sh](../../pkg/webhook/cache/scripts/configure-containerd-registries.sh) script to the OperatingSystemConfig files. The script accepts registries in the format `<upstream_host>,<registry_cache_endpoint>,<upstream_url>` separated by a space. For each given registry, the script waits until the given registry is available (a request to the `<registry_cache_endpoint>` succeeds). Then it creates a `hosts.toml` file for the given `<upstream_host>`. In short, the `hosts.toml` file instructs containerd to first try to pull images for the given `<upstream_host>` from the configured `<registry_cache_endpoint>`. For more information about containerd registry configuration, see the [containerd documentation](https://github.com/containerd/containerd/blob/main/docs/hosts.md). The motivation to introduce the `configure-containerd-registries.sh` script is that we need to create the `hosts.toml` file when the corresponding registry is available. For more details, see [Issue #68 at gardener/gardener-extension-registry-cache](https://github.com/gardener/gardener-extension-registry-cache/pull/68).
   1. The webhook appends the `configure-containerd-registries.service` unit to the OperatingSystemConfig units. The webhook fetches the Extension resource, and then it configures the unit to invoke the `configure-containerd-registries.sh` script with the registries from the Extension status.

### Extension Disablement

This section outlines how the extension disablement works, i.e., the extension has to be removed from the shoot spec.

1. As part of the shoot reconciliation flow, the gardenlet destroys the [Extension](https://github.com/gardener/gardener/blob/master/docs/extensions/extension.md) resource because it is no longer needed.
   1. If the Extension resource contains registries in its status, the registry-cache extension deploys the [`registry-cleaner` DaemonSet](../../pkg/component/registryconfigurationcleaner/registry_configuration_cleaner.go) to the shoot cluster to clean up the existing registry configuration.
   1. The extension deletes the ManagedResource containing the registry cache resources.

## Shoot Deletion

This section outlines how the deletion works for a shoot with the registry-cache extension enabled.

1. As part of the shoot deletion flow, the gardenlet destroys the [Extension](https://github.com/gardener/gardener/blob/master/docs/extensions/extension.md) resource.
   1. In the shoot deletion flow, the Extension resource is deleted after the Worker resource. Hence, there is no need to deploy the [`registry-cleaner` DaemonSet](../../pkg/component/registryconfigurationcleaner/registry_configuration_cleaner.go) to the shoot cluster to clean up the existing registry configuration.
   1. The extension deletes the ManagedResource containing the registry cache resources.
