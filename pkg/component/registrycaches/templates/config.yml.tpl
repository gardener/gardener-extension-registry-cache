# Maintain this file with the default config file (/etc/distribution/config.yml) from the registry image (europe-docker.pkg.dev/gardener-project/releases/3rd/registry:3.0.0-beta.1).
version: 0.1
log:
  fields:
    service: registry
storage:
  delete:
    enabled: true
  # Mitigate https://github.com/distribution/distribution/issues/2367 by disabling the blobdescriptor cache.
  # For more details, see https://github.com/distribution/distribution/issues/2367#issuecomment-1874449361.
  # cache:
  #  blobdescriptor: inmemory
  filesystem:
    rootdirectory: /var/lib/registry
  tag:
    concurrencylimit: 5
http:
  addr: {{ .http_addr }}
  debug:
    addr: {{ .http_debug_addr }}
    prometheus:
      enabled: true
      path: /metrics
  draintimeout: 25s
  tls:
    certificate: /etc/docker/registry/certs/tls.crt
    key: /etc/docker/registry/certs/tls.key
  headers:
    X-Content-Type-Options: [nosniff]
health:
  storagedriver:
    enabled: true
    interval: 10s
    threshold: 3
proxy:
  remoteurl: {{ .proxy_remoteurl }}
  ttl: {{ .proxy_ttl }}
  {{- if and .proxy_username .proxy_password }}
  username: {{ .proxy_username }}
  password: '{{ .proxy_password }}'
  {{- end }}
