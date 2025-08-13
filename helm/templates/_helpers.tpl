{{/*
Expand the helper functions for the fatnotificationcat
*/}}

{{- define "fatnotificationcat.fullname" -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "fatnotificationcat.redis-name" -}}
{{- include "fatnotificationcat.fullname" . }}-redis
{{- end -}}

{{- define "fatnotificationcat.labels" -}}
app: {{ .Chart.Name }}
release: {{ .Release.Name }}
{{- end -}}

{{- define "fatnotificationcat.app-image" -}}
{{ .Values.app.image.repository }}:{{ .Values.app.image.tag | default .Chart.AppVersion }}
{{- end -}}
