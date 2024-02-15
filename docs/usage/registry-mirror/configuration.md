# Configuring the Registry Mirror Extension

## Introduction

### Use-case

containerd allows registry mirrors to be configured. Use-cases are:
- Usage of public mirror(s) - for example circumvent issues with the upstream registry such as rate limiting, outages and others.
- Usage of private mirror(s) - for example reduce network costs by using a private mirror running in the same network.

### Solution

The registry-mirror extension allows registry mirror configuration to be configured via the Shoot spec directly.

### How does it work?

When the extension is enabled, the containerd daemon on the Shoot cluster Nodes gets configured to use as a mirror the requested mirrors. For example, if for upstream `docker.io` the mirror `https://mirror.gcr.io` is configured in the Shoot spec, then containerd gets configured to first pull the image from the mirror (`https://mirror.gcr.io` in that case). If this image pull operation fails, containerd falls back to the upstream itself (`docker.io` in that case).

The extension is based on the contract described in [`containerd` Registry Configuration](https://github.com/gardener/gardener/blob/v1.87.0/docs/usage/containerd-registry-configuration.md). The corresponding upstream documentation in containerd is [Registry Configuration - Introduction](https://github.com/containerd/containerd/blob/v1.7.0/docs/hosts.md).

## Shoot Configuration

The Shoot specification has to be adapted to include the `registry-mirror` extension configuration.

Below is an example of `registry-mirror` extension configuration as part of the Shoot spec:

```yaml
apiVersion: core.gardener.cloud/v1beta1
kind: Shoot
metadata:
  name: crazy-botany
  namespace: garden-dev
spec:
  extensions:
  - type: registry-mirror
    providerConfig:
      apiVersion: mirror.extensions.gardener.cloud/v1alpha1
      kind: MirrorConfig
      mirrors:
      - upstream: docker.io
        hosts:
        - host: "https://mirror.gcr.io"
          capabilities: ["pull"]
```

The `providerConfig` field is required.

The `providerConfig.mirrors` field contains information about the registry mirrors to configure. It is a required field. At least one mirror has to be specified.

The `providerConfig.mirror[].upstream` field is the remote registry host to mirror. It is a required field.
The value must be a valid DNS subdomain (RFC 1123). It must not include a scheme or port.

The `providerConfig.mirror[].hosts` field represents the mirror hosts to be used for the upstream. At least one mirror host has to be specified.

The `providerConfig.mirror[].hosts[].host` field is the mirror host. It is a required field.
The value must include a scheme - `http://` or `https://`.

The `providerConfig.mirror[].hosts[].capabilities` field represents the operations a host is capable of performing. This also represents the set of operations for which the mirror host may be trusted to perform. Defaults to `["pull"]`. The supported values are `pull` and `resolve`.
See the [capabilities field documentation](https://github.com/containerd/containerd/blob/v1.7.0/docs/hosts.md#capabilities-field) for more information which operations are considered trusted ones against public/private mirrors.
