{{- define "name" -}}
gardener-extension-registry-cache
{{- end -}}

{{- define "config" -}}
apiVersion: config.registry.extensions.gardener.cloud/v1alpha1
kind: Configuration
{{- end }}

{{-  define "image" -}}
  {{- if hasPrefix "sha256:" .Values.image.tag }}
  {{- printf "%s@%s" .Values.image.repository .Values.image.tag }}
  {{- else }}
  {{- printf "%s:%s" .Values.image.repository .Values.image.tag }}
  {{- end }}
{{- end }}

{{- define "leaderelectionid" -}}
extension-registry-cache-leader-election
{{- end -}}
