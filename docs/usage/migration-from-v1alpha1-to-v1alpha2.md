# Migration from `v1alpha1` to `v1alpha2`

This document descibres how to migrate from API version `registry.extensions.gardener.cloud/v1alpha1` of the `RegistryConfig` to `registry.extensions.gardener.cloud/v1alpha2`.

The `registry.extensions.gardener.cloud/v1alpha1` is deprecated and will be removed in a future version. Use `registry.extensions.gardener.cloud/v1alpha2` instead.

Let's first inspect how the `RegistryConfig` looks like in API version `registry.extensions.gardener.cloud/v1alpha1`:

```yaml
apiVersion: registry.extensions.gardener.cloud/v1alpha1
kind: RegistryConfig
caches:
- upstream: docker.io
  size: 10Gi
  garbageCollection:
    enabled: true
  secretReferenceName: docker-credentials
```

The translation of the above `RegistryConfig` in API version `registry.extensions.gardener.cloud/v1alpha2` is:

```yaml
apiVersion: registry.extensions.gardener.cloud/v1alpha2
kind: RegistryConfig
caches:
- upstream: docker.io
  volume:
    size: 10Gi
    storageClassName: default
  garbageCollection:
    enabled: true
  secretReferenceName: docker-credentials
```

As you can notice, there is one breaking change in API version `registry.extensions.gardener.cloud/v1alpha2` - the `caches[].size` field is moved to `caches[].volume.size`.

`registry.extensions.gardener.cloud/v1alpha2` also adds a new field `caches[].volume.storageClassName`. In `v1alpha1` the StorageClass name was not configurable and the registry-cache extension assumed the StorageClass name to be `default`. When migrating from `v1alpha1` to `v1alpha2`, the `caches[].volume.storageClassName` field has to be set to `default`. This is required due to backwards-compatibility reasons for registry caches created according to the `v1alpha1` API version.
