{{/*
Expand the name of the chart.
*/}}
{{- define "image-preheat.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "image-preheat.fullname" -}}
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
{{- define "image-preheat.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "image-preheat.labels" -}}
helm.sh/chart: {{ include "image-preheat.chart" . }}
{{ include "image-preheat.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "image-preheat.selectorLabels" -}}
app.kubernetes.io/name: {{ include "image-preheat.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "image-preheat.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "image-preheat.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the config map for image list
*/}}
{{- define "image-preheat.imageListConfigMapName" -}}
{{- printf "%s-image-list" (include "image-preheat.fullname" .) }}
{{- end }}

{{/*
Create the name of the config map for distributed lock
*/}}
{{- define "image-preheat.lockConfigMapName" -}}
{{- printf "%s-lock" (include "image-preheat.fullname" .) }}
{{- end }}

{{/*
Create the name of the headless service
*/}}
{{- define "image-preheat.headlessServiceName" -}}
{{- printf "%s-peers" (include "image-preheat.fullname" .) }}
{{- end }} 