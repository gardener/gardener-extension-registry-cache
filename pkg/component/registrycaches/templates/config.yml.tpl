# Maintain this file with the default config file (/etc/docker/registry/config.yml) from the registry image (europe-docker.pkg.dev/gardener-project/releases/3rd/registry:2.8.3).
version: 0.1
log:
  fields:
    service: registry
storage:
  delete:
    enabled: {{ .storage_delete_enabled }}
  # Mitigate https://github.com/distribution/distribution/issues/2367 by disabling the blobdescriptor cache.
  # For more details, see https://github.com/distribution/distribution/issues/2367#issuecomment-1874449361.
  # cache:
  #  blobdescriptor: inmemory
  filesystem:
    rootdirectory: /var/lib/registry
http:
  addr: {{ .http_addr }}
  debug:
    addr: {{ .http_debug_addr }}
    prometheus:
      enabled: true
      path: /metrics
  headers:
    X-Content-Type-Options: [nosniff]
health:
  storagedriver:
    enabled: true
    interval: 10s
    threshold: 3
proxy:
  remoteurl: {{ .proxy_remoteurl }}
  {{- if and .proxy_username .proxy_password }}
  username: {{ .proxy_username }}
  password: '{{ .proxy_password }}'
  {{- end }}
