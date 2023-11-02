# Developer Docs for Gardener Extension Registry Cache

This document outlines how the Shoot reconciliation and deletion work for a Shoot with the registry-cache extension enabled.

## Shoot reconciliation

This section outlines how the reconciliation works for a Shoot with the registry-cache extension enabled.

1. As part of the Shoot reconciliation flow, gardenlet deploys the [Extension](https://github.com/gardener/gardener/blob/v1.82.0/docs/extensions/extension.md) resource.
1. The registry-cache extension reconciles the Extension resource. [pkg/controller/extension/actuator.go](../../pkg/controller/extension/actuator.go) contains the implementation of the [extension.Actuator](https://github.com/gardener/gardener/blob/v1.82.0/extensions/pkg/controller/extension/actuator.go) interface. The reconciliation of an Extension of type `registry-cache` consists of the following steps:
   1. The extension checks if a registry has been removed (by comparing the status and the spec of the Extension). If an upstream is being removed, then it deploys the [`registry-cleaner` DaemonSet](../../pkg/component/registryconfigurationcleaner/registry_configuration_cleaner.go) to the Shoot cluster to clean up the existing configuration for the upstream that has to be removed.
   1. The registry-cache extension deploys resources to the Shoot cluster via ManagedResource. For every configured upstream it creates a StatefulSet (with PVC), Service and other resources.
   1. It lists all Services from the `kube-system` namespace having the `upstream-host` label. It will return an error (and retry in exponential backoff) until the Services count matches the configured registries count.
   1. When there is a Service created for each configured upstream registry, the registry-cache extension populates the Extension resource status. In the Extension status, for each upstream, it maintains an endpoint (in format `http://<cluster-ip>:5000`) which can be used to access the registry cache from within the Shoot cluster. `<cluster-ip>` is the cluster IP of the registry cache Service. The cluster IP of a Service is assigned by the Kubernetes API server on Service creation.
1. As part of the Shoot reconciliation flow, gardenlet deploys the [OperatingSystemConfig](https://github.com/gardener/gardener/blob/v1.82.0/docs/extensions/operatingsystemconfig.md) resource.
1. The registry-cache extension serves a webhook that mutates the OperatingSystemConfig resource for Shoots having the registry-cache extension enabled (the corresponding namespace gets labeled by gardenlet with `extensions.gardener.cloud/registry-cache=true`). [pkg/webhook/operatingsystemconfig/ensurer.go](../../pkg/webhook/operatingsystemconfig/ensurer.go) contains implementation of the [genericmutator.Ensurer](https://github.com/gardener/gardener/blob/v1.82.0/extensions/pkg/webhook/controlplane/genericmutator/mutator.go) interface.
   1. The webhook appends the [configure-containerd-registries.sh](../../pkg/webhook/operatingsystemconfig/scripts/configure-containerd-registries.sh) script to the OperatingSystemConfig files. The script accepts registries in the format `<upstream_host>,<registry_cache_endpoint>,<upstream_url>` separated by a space. For each given registry the script waits until the given registry is available (a request to the `<registry_cache_endpoint>` succeeds). Then it creates a `hosts.toml` file for the given `<upstream_host>`. In short, the `hosts.toml` file instructs containerd to first try to pull images for the given `<upstream_host>` from the configured `<registry_cache_endpoint>`. For more information about containerd registry configuration, see the [containerd documentation](https://github.com/containerd/containerd/blob/main/docs/hosts.md). The motivation to introduce the `configure-containerd-registries.sh` script is that we need to create the `hosts.toml` file when the corresponding registry is available. For more details, see https://github.com/gardener/gardener-extension-registry-cache/pull/68.
   1. The webhook appends the `configure-containerd-registries.service` unit to the OperatingSystemConfig units. The webhook fetches the Extension resource and then it configures the unit to invoke the `configure-containerd-registries.sh` script with the registries from the Extension status.

## Shoot deletion

This section outlines how the deletion works for a Shoot with the registry-cache extension enabled.

1. As part of the Shoot deletion flow, gardenlet destroys the [Extension](https://github.com/gardener/gardener/blob/v1.82.0/docs/extensions/extension.md) resource.
   1. If the Extension resource contains registries in its status, the registry-cache extension deploys the [`registry-cleaner` DaemonSet](../../pkg/component/registryconfigurationcleaner/registry_configuration_cleaner.go) to the Shoot cluster to clean up the existing configuration for the registries.
   1. The extension deletes the ManagedResource containing the registry cache resources.
