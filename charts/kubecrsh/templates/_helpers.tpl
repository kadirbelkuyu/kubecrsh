{{/*
Expand the name of the chart.
*/}}
{{- define "kubecrsh.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kubecrsh.fullname" -}}
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
{{- define "kubecrsh.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kubecrsh.labels" -}}
helm.sh/chart: {{ include "kubecrsh.chart" . }}
{{ include "kubecrsh.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kubecrsh.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kubecrsh.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "kubecrsh.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "kubecrsh.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the secret to use
*/}}
{{- define "kubecrsh.secretName" -}}
{{- if .Values.secrets.existingSecret }}
{{- .Values.secrets.existingSecret }}
{{- else }}
{{- printf "%s-secrets" (include "kubecrsh.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create the name of the configmap
*/}}
{{- define "kubecrsh.configMapName" -}}
{{- printf "%s-config" (include "kubecrsh.fullname" .) }}
{{- end }}

{{/*
Return the target Kubernetes namespace
*/}}
{{- define "kubecrsh.namespace" -}}
{{- default .Release.Namespace .Values.global.namespaceOverride }}
{{- end }}

{{/*
Return the image name with proper tag
*/}}
{{- define "kubecrsh.image" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
Return true if secrets should be created
*/}}
{{- define "kubecrsh.createSecret" -}}
{{- if and .Values.secrets.create (not .Values.secrets.existingSecret) }}
{{- true }}
{{- end }}
{{- end }}

{{/*
Return true if any notifier is enabled
*/}}
{{- define "kubecrsh.notifiersEnabled" -}}
{{- if or .Values.notifiers.slack.enabled .Values.notifiers.webhook.enabled }}
{{- true }}
{{- end }}
{{- end }}

{{/*
Generate daemon args
*/}}
{{- define "kubecrsh.daemonArgs" -}}
- daemon
- --config=/config/config.yaml
- --http-addr={{ .Values.daemon.httpAddr }}
{{- if .Values.notifiers.slack.enabled }}
- --slack-webhook=$(SLACK_WEBHOOK)
{{- end }}
{{- if .Values.notifiers.webhook.enabled }}
- --webhook-url=$(WEBHOOK_URL)
{{- if .Values.secrets.webhookToken }}
- --webhook-token=$(WEBHOOK_TOKEN)
{{- end }}
{{- end }}
{{- range .Values.daemon.extraArgs }}
- {{ . | quote }}
{{- end }}
{{- end }}

{{/*
Generate watch namespace argument
*/}}
{{- define "kubecrsh.watchNamespace" -}}
{{- if .Values.config.namespace }}
{{- .Values.config.namespace }}
{{- else if not .Values.rbac.clusterWide }}
{{- .Release.Namespace }}
{{- else }}
{{- "" }}
{{- end }}
{{- end }}
