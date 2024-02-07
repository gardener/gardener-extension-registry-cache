server = "{{ .Server }}"
{{ range $registryHost := .Hosts }}
[host."{{ $registryHost.Host }}"]
  capabilities = {{ $registryHost.Capabilities | toJson | replace "," ", " }}
{{ end }}