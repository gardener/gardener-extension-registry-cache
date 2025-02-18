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

{{-  define "image" -}}
  {{- if .Values.image.ref -}}
  {{ .Values.image.ref }}
  {{- else -}}
  {{- if hasPrefix "sha256:" .Values.image.tag }}
  {{- printf "%s@%s" .Values.image.repository .Values.image.tag }}
  {{- else }}
  {{- printf "%s:%s" .Values.image.repository .Values.image.tag }}
  {{- end }}
  {{- end -}}
{{- end }}
