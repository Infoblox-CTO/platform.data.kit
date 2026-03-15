{{/*
Common labels for dk-app resources.
*/}}
{{- define "dk-app.labels" -}}
app.kubernetes.io/name: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: dk
app.kubernetes.io/version: {{ .Values.appVersion | quote }}
{{- end }}
