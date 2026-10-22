{{- define "orders.name" -}}orders{{- end }}
{{- define "orders.fullname" -}}{{ include "orders.name" . }}{{- end }}
{{- define "orders.labels" -}}
app.kubernetes.io/name: {{ include "orders.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: streamweave
{{- end }}
{{- define "orders.serviceAccountName" -}}{{ default (include "orders.fullname" .) .Values.serviceAccount.name }}{{- end }}
{{- define "orders.apiServiceAccountName" -}}{{ default (printf "%s-api" (include "orders.fullname" .)) .Values.serviceAccount.apiName }}{{- end }}
{{- define "orders.relayServiceAccountName" -}}{{ default (printf "%s-relay" (include "orders.fullname" .)) .Values.serviceAccount.relayName }}{{- end }}
{{- define "orders.image" -}}
{{- if and .Values.image.requireDigest (not .Values.image.digest) -}}
{{- fail "orders.image.digest is required for this environment" -}}
{{- end -}}
{{- .Values.image.repository }}{{ if .Values.image.digest }}@{{ .Values.image.digest }}{{ else }}:{{ .Values.image.tag }}{{ end }}
{{- end }}
