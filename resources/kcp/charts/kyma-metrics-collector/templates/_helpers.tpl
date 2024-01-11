{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "kyma-metrics-collector.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kyma-metrics-collector.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/* Create chart name and version as used by the chart label. */}}
{{- define "kyma-metrics-collector.chartref" -}}
{{- replace "+" "_" .Chart.Version | printf "%s-%s" .Chart.Name -}}
{{- end }}

{{/* Generate basic labels */}}
{{- define "kyma-metrics-collector.labels" -}}
chart: {{ template "kyma-metrics-collector.chartref" . }}
release: {{ .Release.Name | quote }}
heritage: {{ .Release.Service | quote }}
{{- end }}

{{- define "kyma-metrics-collector.prometheusrule.labels" -}}
chart: {{ template "kyma-metrics-collector.chartref" . }}
release: {{ .Values.prometheus.labels.release }}
app: {{ .Values.prometheus.labels.app }}
heritage: {{ .Release.Service | quote }}
{{- end }}

{{- define "kyma-metrics-collector.publicCloud.configMap.labels" -}}
{{ template "kyma-metrics-collector.labels" . }}
user-by: {{ template "kyma-metrics-collector.name" . }}
{{- end }}

{{- define "kyma-metrics-collector.publicCloud.configMap.name" -}}
{{ printf "%s-%s" (include "kyma-metrics-collector.fullname" .) "public-cloud-spec" }}
{{- end -}}

{{- define "kyma-metrics-collector.imagePullSecrets" -}}
{{- if .Values.image.pullSecrets }}
imagePullSecrets:
{{- range .Values.image.pullSecrets }}
  - name: {{ . }}
{{- end }}
{{- else if .Values.global }}
{{- if .Values.global.imagePullSecrets }}
imagePullSecrets:
{{- range .Values.global.imagePullSecrets }}
  - name: {{ . }}
{{- end }}
{{- end -}}
{{- end -}}
{{- end -}}


{{- define "kyma-metrics-collector.image" -}}
{{- $repository := "" -}}
{{- $tag := "" -}}
{{- if .Values.global -}}
  {{- if .Values.global.images -}}
    {{- if .Values.global.images.containerRegistry -}}
      {{- $repository = printf "%s/%skyma-metrics-collector" .Values.global.images.containerRegistry.path (default "" .Values.global.images.kyma_metrics_collector.dir) -}}
      {{- $tag = .Values.global.images.kyma_metrics_collector.version | toString -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{- if .Values.image -}}
{{- if .Values.image.repository -}}
{{- $repository = .Values.image.repository -}}
{{- end -}}
{{- if .Values.image.tag -}}
{{- $tag = .Values.image.tag | toString -}}
{{- end -}}
{{- end -}}
{{- printf "%s:%s" $repository $tag -}}
{{- end -}}
