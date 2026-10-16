{{/*
Purpose: Renders the main vLLM inference container, including model arguments,
ports, environment, probes, GPU resources, and mounts. Receives the chart root
and the storage plan from deployment.yaml; creates no standalone resources.
*/}}
{{- define "vllm.inferenceContainer" -}}
{{- $root := .root -}}
{{- $values := $root.Values -}}
{{- $environmentSources := list -}}
{{- if $values.environmentVariable.enabled -}}
  {{- range $values.environmentVariable.configMaps -}}
    {{- $environmentSources = append $environmentSources (dict "configMapRef" (dict "name" .)) -}}
  {{- end -}}
  {{- range $values.environmentVariable.secrets -}}
    {{- $environmentSources = append $environmentSources (dict "secretRef" (dict "name" .)) -}}
  {{- end -}}
{{- end -}}
name: {{ include "vllm.fullname" $root }}
image: {{ include "vllm.image" $root }}
imagePullPolicy: {{ $values.image.imagePullPolicy }}
{{- with $values.image.command }}
command:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with $values.image.args }}
args:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with $values.securityContext }}
securityContext:
  {{- toYaml . | nindent 2 }}
{{- end }}
ports:
  - name: {{ $values.image.containerPortName | default "http" }}
    containerPort: {{ $values.image.containerPort }}
    protocol: {{ $values.image.portProtocol | default "TCP" }}
  {{- if $values.additionalContainerPorts.enabled }}
  {{- range $values.additionalContainerPorts.additionalPorts }}
  - name: {{ .portName }}
    containerPort: {{ .containerPort }}
    protocol: {{ .protocol | default "TCP" }}
  {{- end }}
  {{- end }}
{{- with $values.env }}
env:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with $environmentSources }}
envFrom:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{/* Strip the chart's enable switch; pass every configured probe field through. */}}
{{- range $name := list "startupProbe" "readinessProbe" "livenessProbe" }}
{{- $probe := index $values $name }}
{{- if $probe.enabled }}
{{ $name }}:
  {{- omit $probe "enabled" | toYaml | nindent 2 }}
{{- end }}
{{- end }}
resources:
  {{- toYaml $values.resources | nindent 2 }}
{{- with .storage.mounts }}
volumeMounts:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- end -}}
