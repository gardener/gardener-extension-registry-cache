server = "{{ .Server }}"
{{ range $registryHost := .Hosts }}
[host."{{ $registryHost.Host }}"]
{{ end }}