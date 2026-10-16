{{/*
Purpose: Builds pod volumes and inference-container mounts together so model
cache and configuration references stay aligned. Custom volumes and mounts
remain independent to support init containers and sidecars. Creates no resources.
*/}}
{{- define "vllm.storage" -}}
{{- $volumes := list -}}
{{- $mounts := list -}}

{{/* Existing and chart-managed model caches use the same pod wiring. */}}
{{- if .Values.pvc.enabled -}}
  {{- range .Values.pvc.volumes -}}
    {{- $volumes = append $volumes (dict "name" .volumeName "persistentVolumeClaim" (dict "claimName" .claimName)) -}}
    {{- $mounts = append $mounts (dict "name" .volumeName "mountPath" .mountPath "readOnly" (.readOnly | default false)) -}}
  {{- end -}}
{{- end -}}

{{/* ConfigMaps and Secrets share mount settings but use different volume keys. */}}
{{- if .Values.configurationVolume.enabled -}}
  {{- range $source := list
      (dict "list" "configMaps" "kind" "configMap" "reference" "configMapName" "field" "name")
      (dict "list" "secrets" "kind" "secret" "reference" "secretName" "field" "secretName") -}}
    {{- range index $.Values.configurationVolume $source.list -}}
      {{- $reference := dict $source.field (index . $source.reference) -}}
      {{- $volumes = append $volumes (dict "name" .volumeName $source.kind $reference) -}}
      {{- $mounts = append $mounts (dict "name" .volumeName "mountPath" .mountPath "readOnly" (.readOnly | default false)) -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{- if .Values.customVolumes.enabled -}}
  {{- $volumes = concat $volumes (.Values.customVolumes.volumes | default list) -}}
{{- end -}}
{{- if .Values.customVolumeMountforMainContainer.enabled -}}
  {{- $mounts = concat $mounts (.Values.customVolumeMountforMainContainer.volumeMounts | default list) -}}
{{- end -}}
{{- toYaml (dict "volumes" $volumes "mounts" $mounts) -}}
{{- end -}}
