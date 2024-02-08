# Migration from `v1alpha2` to `v1alpha3`

This document descibres how to migrate from API version `registry.extensions.gardener.cloud/v1alpha2` of the `RegistryConfig` to `registry.extensions.gardener.cloud/v1alpha3`.

The `registry.extensions.gardener.cloud/v1alpha2` is deprecated and will be removed in a future version. Use `registry.extensions.gardener.cloud/v1alpha3` instead.

Let's first inspect how the `RegistryConfig` looks like in API version `registry.extensions.gardener.cloud/v1alpha2`:

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

The translation of the above `RegistryConfig` in API version `registry.extensions.gardener.cloud/v1alpha3` is:

```yaml
apiVersion: registry.extensions.gardener.cloud/v1alpha3
kind: RegistryConfig
caches:
- upstream: docker.io
  volume:
    size: 10Gi
    storageClassName: default
  garbageCollection:
    ttl: 168h
  secretReferenceName: docker-credentials
```

As you can notice, there is one breaking change in API version `registry.extensions.gardener.cloud/v1alpha3` - the `caches[].garbageCollection.enabled` field is replaced by `caches[].garbageCollection.ttl`.

`v1alpha2` was an API created against `distribution/distribution@2.8`. The registry-cache evolves and upgraded recently to `distribution/distribution@3.0`. In `distribution/distribution@2.8` the ttl is not configurable and it is hard-coded to `168h` (7 days). That's why in versions prior to `v1alpha3` the `storage.delete.enabled` config field in the `distribution/distribution` configuration was used to control deletion of blobs. `distribution/distribution@3.0` exposes the `proxy.ttl` config field. It is now possible to natively disable the garbage collection (expiration of blobs) by setting `proxy.ttl=0`.

Conversions are as follows:
- `garbageCollection.enable=true` gets converted to `garbageCollection.ttl=168h` and vice versa (a positive `garbageCollection.ttl` duration is considered `garbageCollection.enable=true`)
- `garbageCollection.enable=false` gets converted to `garbageCollection.ttl=0` vice versa
