# How to provide credentials for upstream registry?

In order to pull private images through registry cache, it is required to supply credentials for the private upstream registry.

## How to configure the registry cache to use upstream registry credentials?

1. Create an immutable Secret with the upstream registry credentials in the Garden cluster

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

1. Add the newly created Secret as a reference to the Shoot spec, and then to the registry-cache extension configuration

   In the registry-cache configuration set the `secretReferenceName` field. It should point to a resource reference under `spec.resources`. The resource reference itself points to the Secret in project namespace.

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
1. Generate new pair of credentials in the cloud provider account. Do not invalidate the old ones.
1. Create a new Secret (e.g. `ro-docker-secret-v2`) with the newly generated credentials as described step 1. in [How to configure the registry cache to use upstream registry credentials?](#how-to-configure-the-registry-cache-to-use-upstream-registry-credentials).
1. Update the Shoot spec with newly created Secret as described step 2. in [How to configure the registry cache to use upstream registry credentials?](#how-to-configure-the-registry-cache-to-use-upstream-registry-credentials).
1 The above step will trigger a Shoot reconciliation. Wait for the Shoot reconciliation to complete.
1. Make sure that the old Secret is no longer referenced by any Shoot cluster. Finally, delete the Secret containing the old credentials (e.g. `ro-docker-secret-v1`).
1. Delete the corresponding old credentials from the cloud provider account.

## Gotchas

- The registry cache provides the credentials for every request against the corresponding upstream. In some cases, misconfigured credentials can prevent the registry cache to pull even public images from the upstream (for example: invalid service account key for Artifact Registry). However, this behaviour is controlled by the server-side logic of the upstream registry.
