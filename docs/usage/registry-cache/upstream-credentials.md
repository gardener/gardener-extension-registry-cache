---
title: How to provide credentials for upstream registry?
---

# How to provide credentials for upstream registry?

In Kubernetes, to pull images from private container image registries you either have to specify an image pull secret (see [Pull an Image from a Private Registry](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)) or you have to configure the kubelet to dynamically retrieve credentials using a credential provider plugin (see [Configure a kubelet Image Credential Provider](https://kubernetes.io/docs/tasks/administer-cluster/kubelet-credential-provider/)). When pulling an image, the kubelet is providing the credentials to the CRI implementation. The CRI implementation uses the provided credentials against the upstream registry to pull the image.

The registry-cache extension is using the [Distribution project](https://github.com/distribution/distribution) as pull through cache implementation. The Distribution project does not use the provided credentials from the CRI implementation while fetching an image from the upstream. Hence, the above-described scenarios such as configuring image pull secret for a pod or configuring kubelet credential provider plugins don't work out of the box with the pull through cache provided by the registry-cache extension.
Instead, the Distribution project supports configuring only one set of credentials for a given pull through cache instance (for a given upstream).

This document describe how to supply credentials for the private upstream registry in order to pull private image with the registry cache.

## How to configure the registry cache to use upstream registry credentials?

1. Create an immutable secret with the upstream registry credentials in the Garden cluster:

   ```bash
   % kubectl create -f - <<EOF
   apiVersion: v1
   kind: Secret
   metadata:
     name: ro-docker-secret-v1
     namespace: garden-dev
   type: Opaque
   immutable: true
   data:
     username: $(echo -n $USERNAME | base64 -w0)
     password: $(echo -n $PASSWORD | base64 -w0)
   EOF
   ```

   For Artifact Registry, the username is `_json_key` and the password is the service account key in JSON format. To base64 encode the service account key, copy it and run:

   ```bash
   % echo -n $SERVICE_ACCOUNT_KEY_JSON | base64 -w0
   ```

1. Add the newly created secret as a reference to the shoot spec, and then to the registry-cache extension configuration.

   In the registry-cache configuration, set the `secretReferenceName` field. It should point to a resource reference under `spec.resources`. The resource reference itself points to the secret in project namespace.

   ```yaml
   apiVersion: core.gardener.cloud/v1beta1
   kind: Shoot
   # ...
   spec:
     extensions:
     - type: registry-cache
       providerConfig:
         apiVersion: registry.extensions.gardener.cloud/v1alpha3
         kind: RegistryConfig
         caches:
         - upstream: docker.io
           secretReferenceName: docker-secret
     # ...
     resources:
     - name: docker-secret
       resourceRef:
         apiVersion: v1
         kind: Secret
         name: ro-docker-secret-v1
   # ...
   ```

## How to rotate the registry credentials?

To rotate registry credentials perform the following steps:
1. Generate a new pair of credentials in the cloud provider account. Do not invalidate the old ones.
1. Create a new secret (e.g., `ro-docker-secret-v2`) with the newly generated credentials as described in step 1. in [How to configure the registry cache to use upstream registry credentials?](#how-to-configure-the-registry-cache-to-use-upstream-registry-credentials).
1. Update the shoot spec with newly created secret as described in step 2. in [How to configure the registry cache to use upstream registry credentials?](#how-to-configure-the-registry-cache-to-use-upstream-registry-credentials).
1. The above step will trigger a shoot reconciliation. Wait for it to complete.
1. Make sure that the old secret is no longer referenced by any shoot cluster. Finally, delete the secret containing the old credentials (e.g., `ro-docker-secret-v1`).
1. Delete the corresponding old credentials from the cloud provider account.

## Possible Pitfalls

- The registry cache is not protected by any authentication/authorization mechanism. The cached images (incl. private images) can be fetched from the registry cache without authentication/authorization. Note that the registry cache itself is not exposed publicly.
- The registry cache provides the credentials for every request against the corresponding upstream. In some cases, misconfigured credentials can prevent the registry cache to pull even public images from the upstream (for example: invalid service account key for Artifact Registry). However, this behaviour is controlled by the server-side logic of the upstream registry.
