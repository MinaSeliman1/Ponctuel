{{- define "ponctuel.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "ponctuel.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name (include "ponctuel.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{- define "ponctuel.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
app.kubernetes.io/name: {{ include "ponctuel.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "ponctuel.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ponctuel.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "ponctuel.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "ponctuel.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "ponctuel.secretName" -}}
{{- printf "%s-secrets" (include "ponctuel.fullname" .) }}
{{- end }}

{{- define "ponctuel.configMapName" -}}
{{- printf "%s-config" (include "ponctuel.fullname" .) }}
{{- end }}
