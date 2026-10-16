{{/*
Purpose: Defines reusable vllm.* helpers for resource names, labels, image references,
and service-account selection. Other templates include these helpers to keep
resource metadata and pod selectors consistent; this file creates no resources.
*/}}
{{/*
vLLM workload name.
*/}}
{{- define "vllm.name" -}}
{{- default .Chart.Name .Values.microserviceName | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Resource name; retain microserviceName for existing Deployment adoption.
*/}}
{{- define "vllm.fullname" -}}
{{- if .Values.microserviceName -}}
{{- .Values.microserviceName | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := .Chart.Name -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Chart version label.
*/}}
{{- define "vllm.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Preserve the selector of the existing vLLM Deployment.

Do not change this selector during Helm adoption.
*/}}
{{- define "vllm.selectorLabels" -}}
app: {{ include "vllm.name" . }}
{{- end -}}

{{/*
Combine user labels with the workload identity in one map. Chart-owned labels
take precedence so custom labels cannot break the Deployment/Service selector.
*/}}
{{- define "vllm.labels" -}}
{{- $identity := dict
    "helm.sh/chart" (include "vllm.chart" .)
    "app" (include "vllm.name" .)
    "app.kubernetes.io/name" (include "vllm.name" .)
    "app.kubernetes.io/instance" .Release.Name
    "app.kubernetes.io/managed-by" .Release.Service
    "app.kubernetes.io/component" "inference" -}}
{{- with .Chart.AppVersion -}}
  {{- $_ := set $identity "app.kubernetes.io/version" . -}}
{{- end -}}
{{- mergeOverwrite (deepCopy (.Values.commonLabels | default dict)) $identity | toYaml -}}
{{- end -}}

{{/*
Image resolution.

Supports:
1. Complete image URL
2. Legacy global registry (backward compatible)
3. Workload-specific registry
4. Direct ImagePath
*/}}
{{- define "vllm.image" -}}
{{- $image := .Values.image -}}
{{- if $image.ImageFullurl -}}
{{- $image.ImageFullurl | quote -}}
{{- else -}}
{{- $path := required "image.ImagePath or image.ImageFullurl is required" $image.ImagePath -}}
{{- $globalRegistry := dig "enterpriseCharts" "image" "registryDomain" "" (.Values.global | default dict) -}}
{{- $registry := coalesce $globalRegistry $image.registryDomain -}}
{{- if $registry -}}
{{- printf "%s/%s" $registry $path | quote -}}
{{- else -}}
{{- $path | quote -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Service account name.
*/}}
{{- define "vllm.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "vllm.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}
