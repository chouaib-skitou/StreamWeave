{{- define "identity.name" -}}identity{{- end -}}
{{- define "identity.fullname" -}}identity{{- end -}}
{{- define "identity.secretName" -}}
{{- if .Values.secret.existingSecret -}}
{{- .Values.secret.existingSecret -}}
{{- else if .Values.secret.create -}}
{{- printf "%s-secret" (include "identity.fullname" .) -}}
{{- else -}}
{{- required "secret.existingSecret is required when secret.create is false" .Values.secret.existingSecret -}}
{{- end -}}
{{- end -}}
