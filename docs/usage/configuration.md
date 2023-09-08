# Configuring the Registry Cache Extension

## Introduction

### Use-case

For a Shoot cluster, the containerd daemon of every Node goes to the internet and fetches an image that it doesn't have locally in the Node's image cache. New Nodes are often created due to events such as auto-scaling (scale up), rolling update, or replacement of unhealthy Node. Such a new Node would need to pull all of the images of the Pods running on it from the internet because the Node's cache is initially empty. Pulling an image from a registry produces network traffic and registry costs. To avoid these network traffic and registry costs, you can use the registry-cache extension to run a registry as pull-through cache.

The following diagram shows a rough outline of how image pull looks like for a Shoot cluster **without registry cache**:
![shoot-cluster-without-registry-cache](./images/shoot-cluster-without-registry-cache.png)

### Solution

The registry-cache extension deploys and manages a registry in the Shoot cluster that runs as pull-through cache. The used registry implementation is [distribution/distribution](https://github.com/distribution/distribution).

### How does it work?

When the extension is enabled, a registry cache for each configured upstream is deployed to the Shoot cluster. Along with this, the containerd daemon on the Shoot cluster Nodes gets configured to use as a mirror the Service IP address of the deployed registry cache. For example, if a registry cache for upstream `docker.io` is requested via the Shoot spec, then containerd gets configured to first pull the image from the deployed cache in the Shoot cluster. If this image pull operation fails, containerd falls back to the upstream itself (`docker.io` in that case).

The first time an image is requested from the pull-through cache, it pulls the image from the configured upstream registry and stores it locally before handing it back to the client. On subsequent requests, the pull-through cache is able to serve the image from its own storage.

> Note: The used registry implementation ([distribution/distribution](https://github.com/distribution/distribution)) supports mirroring of only one upstream registry.

The following diagram shows a rough outline of how image pull looks like for a Shoot cluster **with registry cache**:
![shoot-cluster-with-registry-cache](./images/shoot-cluster-with-registry-cache.png)

## Shoot Configuration

The extension is not globally enabled and must be configured per Shoot cluster. The Shoot specification has to be adapted to include the `registry-cache` extension configuration.

Below is an example of `registry-cache` extension configuration as part of the Shoot spec:

```yaml
apiVersion: core.gardener.cloud/v1beta1
kind: Shoot
metadata:
  name: crazy-botany
  namespace: garden-dev
spec:
  extensions:
  - type: registry-cache
    providerConfig:
      apiVersion: registry.extensions.gardener.cloud/v1alpha1
      kind: RegistryConfig
      caches:
      - upstream: docker.io
        size: 100Gi
      - upstream: ghcr.io
      - upstream: quay.io
        garbageCollection:
          enabled: false
```

The `providerConfig` field is required.

The `providerConfig.caches` field contains information about the registry caches to deploy. It is a required field. At least one cache has to be specified.

The `providerConfig.caches[].upstream` field is the remote registry host (and optionally port) to cache. It is a required field.
The desired format is `host[:port]`. The value must not include a scheme. The configured upstream registry must be accessible by `https` (`https://` is the assumed scheme).

The `providerConfig.caches[].size` field is the size of the registry cache. Defaults to `10Gi`. The size must be a positive quantity (greater than 0).
The registry-cache extension deploys a StatefulSet with a volume claim template. A PersistentVolumeClaim is created with the default StorageClass and the configured size.

The `providerConfig.caches[].garbageCollection.enabled` field enables/disables the cache's garbage collection. Defaults to `true`. The time to live (ttl) for an image is `7d`. See the [garbage collection section](#garbage-collection) for more details.

## Garbage Collection

When the registry cache receives a request for an image that is not present in its local store, it fetches the image from the upstream, returns it to the client and stores the image in the local store. The registry cache runs a scheduler that deletes images when their time to live (ttl) expires. When adding an image to the local store, the registry cache also adds a time to live for the image. The ttl value is `7d`.
At the time of writing this document, there is no functionality for garbage collection based on disk size - e.g. garbage collecting images when a certain disk usage threshold is passed.

## Increase the cache disk size

When there is no available disk space, the registry cache continues to respond to requests. However, it cannot store the remotely fetched images locally because it has no free disk space. In such case, it is simply acting as a proxy without being able to cache the images in its local store. The disk has to be resized to ensure that the registry cache continues to cache images.

There are two alternatives to enlarge the cache's disk size.

#### [Alternative 1] Resize the PVC

To enlarge the PVC's size follow the following steps:
1. Make sure that the `KUBECONFIG` environment variable is targeting the correct Shoot cluster.

2. Find the PVC name to resize for the desired upstream. The below example fetches the PVC for the `docker.io` upstream:

   ```
   % kubectl -n kube-system get pvc -l upstream-host=docker.io
   ```

3. Patch the PVC's size to the desired size. The below example patches the size of a PVC to `10Gi`:

   ```
   % kubectl -n kube-system patch pvc $PVC_NAME --type merge -p '{"spec":{"resources":{"requests": {"storage": "10Gi"}}}}'
   ```

4. Make sure that the PVC gets resized. Describe the PVC to check the resize operation result:
   
   ```
   % kubectl -n kube-system describe pvc -l upstream-host=docker.io
   ```

> Drawback of this approach: The cache's size in the Shoot spec (`providerConfig.caches[].size`) diverges from the PVC's size.

#### [Alternative 2] Remove and readd the cache

There is always the option to remove the cache from the Shoot spec and to readd it again with the updated size.

> Drawback of this approach: The already cached images get lost and the cache starts with an empty disk.

## High-availability

The registry cache runs with a single replica. This fact may lead to concerns for the high-availability such as "What happens when the registry cache is down? Does containerd fail to pull the image?". As outlined in the [How does it work? section](#how-does-it-work), containerd is configured to fall back to the upstream registry if it fails to pull the image from the registry cache. Hence, when the registry cache is unavailable, the containerd's image pull operations are not affected because containerd falls back to image pull from the upstream registry.

## Gotchas

- The used registry implementation ([distribution/distribution](https://github.com/distribution/distribution)) supports mirroring of only one upstream registry. The extension deploys a pull-through cache for each configured upstream.
- `gcr.io`, `us.gcr.io`, `eu.gcr.io`, and `asia.gcr.io` are different upstreams. Hence, configuring `gcr.io` as upstream won't cache images from `us.gcr.io`, `eu.gcr.io`, or `asia.gcr.io`.

## Limitations

- A registry cache cannot cache content from the Shoot system components if such upstream is requested:
  - On Shoot creation with the registry cache extension enabled, a registry cache is unable to cache all of images from the Shoot system components because a registry cache Pod requires its PVC to be provisioned, attached and mounted (the corresponding CSI node plugin needs to be running). Usually, until the registry cache Pod is running containerd falls back to the upstream for pulling the images from the Shoot system components.
  - On new Node creation for existing Shoot with the registry cache extension enabled, a registry cache is unable to cache most of the images from  Shoot system components because the containerd registry configuration on that Node is applied after the registry cache Service is reachable from the Node (the `configure-containerd-registries.service` unit is the machinery that does this). The reachability of the registry cache Service requires the Service network to be set up, i.e the kube-proxy for that new Node to be running and to have set up iptables/IPVS configuration for the registry cache Service.
- Services cannot be resolved by DNS from the Node. That's why the registry cache's Service cluster IP is configured in containerd (instead of the Service DNS). A Service's cluster IP is assigned on its creation by the kube-apiserver. Deletion of the registry cache's Service by Shoot owner would lead the Service to be recreated with a new cluster IP. In such case, until the next Shoot reconciliation, containerd will be configured with the old cluster IP. Hence, containerd will fail to pull images from the cache.
- containerd is configured to fall back to the upstream itself if a request against the cache fails. However, if the cluster IP of the registry cache Service does not exist or if kube-proxy hasn't configured iptables/IPVS rules for the registry cache Service, then containerd requests against the registry cache time out in 30 seconds. This increases significantly the image pull times because containerd does multiple requests as part of the image pull (HEAD request to resolve the manifest by tag, GET request for the manifest by SHA, GET requests for blobs).
  - Example: If the Service of a registry cache is deleted, then a new Service will be created. containerd registry config will still contain the old Service's cluster IP.
    - Image pull of `docker.io/library/alpine:3.13.2` from the upstream takes ~2s while image pull of the same image with invalid registry cache cluster IP takes ~2m.2s.
    - Image pull of `eu.gcr.io/gardener-project/gardener/ops-toolbelt:0.18.0` from the upstream takes ~10s while image pull of the same image with invalid registry cache cluster IP takes ~3m.10s.
