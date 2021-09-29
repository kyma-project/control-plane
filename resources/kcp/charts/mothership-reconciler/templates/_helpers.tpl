{{/*
Expand the name of the chart.
*/}}
{{- define "mothership-reconciler.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name, it will be used as the full name.
*/}}
{{- define "mothership-reconciler.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "mothership-reconciler.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "mothership-reconciler.labels" -}}
helm.sh/chart: {{ include "mothership-reconciler.chart" . }}
{{ include "mothership-reconciler.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "mothership-reconciler.selectorLabels" -}}
app.kubernetes.io/name: {{ include "mothership-reconciler.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "mothership-reconciler.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "mothership-reconciler.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "mothership-reconciler.component-reconcilers" -}}
{{- range $component := .Values.global.components }}
  "{{ $component }}": {
    "url": "http://{{ $component }}-reconciler:8080/v1/run"
  },
{{- end }}
{{- end }}
