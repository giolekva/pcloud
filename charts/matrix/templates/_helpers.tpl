{{- define "clientSecret" -}}
{{- if .Values.oauth2.clientSecret -}}
{{- .Values.oauth2.clientSecret -}}
{{- else -}}
{{- randAlphaNum 32 -}}
{{- end -}}
{{- end -}}
