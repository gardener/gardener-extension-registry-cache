{{- define "name" -}}
gardener-extension-registry-cache
{{- end -}}

{{- define "config" -}}
apiVersion: config.registry.extensions.gardener.cloud/v1alpha1
kind: Configuration
{{- end }}

{{- define "leaderelectionid" -}}
extension-registry-cache-leader-election
{{- end -}}
