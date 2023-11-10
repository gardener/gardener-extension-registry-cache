# How to provide credentials for upstream repository

In order to pull private images through registry cache, it is required to supply credentials for private upstream repository.

## Configure registry cache to use upstream repository credentials

### Create a `immutable` Secret with upstream repository credentials
1. Encode credentials with base64 encoding:
     ```bash
     % echo "auser" | base64 -w 0
     YXVzZXIK%
     % echo "asecret" | base64 -w 0
     YXNlY3JldAo=%
     ```
        > **Note** In some cases password can be a json (e.g. _json_key in **gcr** registry).
        > If so, make sure the json value is enclosed in single quotes:
     ```bash
     echo "'{"type": "service_account",... ,"universe_domain": "googleapis.com"}'" | base64 -w 0
     J3t0eXBlOiBzZXJ2aWNlX2FjY291bnQsLi4uICx1bml2ZXJzZV9kb21haW46IGdvb2dsZWFwaXMuY29tfScK%
     ```
2. Use encoded values to build immutable secret in your own project `% kubectl create -f <path_to_secret_yaml>`:
     ```yaml
     apiVersion: v1
     data:
      username: YXVzZXIK
      password: YXNlY3JldAo=
     kind: Secret
     immutable: true
     metadata:
      name: ro-docker-secret
      namespace: garden-dev
     type: Opaque
     ```
### Define *secretReferenceName* field in registry cache config to refer the secret
In the cache configuration define `secretReferenceName`. It should point to a `resourceRef` under `spec.resources` that points to the `secret` in project namespace.

```yaml
apiVersion: core.gardener.cloud/v1beta1
kind: Shoot
#...
spec:
  extensions:
  - type: registry-cache
    providerConfig:
      apiVersion: registry.extensions.gardener.cloud/v1alpha1
      kind: RegistryConfig
      caches:
      - upstream: docker.io
        size: 800Mi
        garbageCollection:
          enabled: true
        secretReferenceName: docker-secret
#...        
  resources:
  - name: docker-secret
    resourceRef:
      apiVersion: v1
      kind: Secret
      name: ro-docker-secret
#...
```
## Rotate repository credentials

To rotate repository credentials perform the following steps:
- Create a secret with new credentials as describe [here](#create-a-immutable-secret-with-upstream-repository-credentials).
- Update Shoot spec with newly created secret as described [here](#define-secretreferencename-field-in-registry-cache-config-to-refer-the-secret).
- Wait for the Shoot reconciliation to complete.
- Delete the secret containing the old credentials.