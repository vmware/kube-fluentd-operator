{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "fluentd-router.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "fluentd-router.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Set apiVersion based on .Capabilities.APIVersions
*/}}
{{- define "rbacAPIVersion" -}}
{{- if .Capabilities.APIVersions.Has "rbac.authorization.k8s.io/v1" -}}
rbac.authorization.k8s.io/v1
{{- else if .Capabilities.APIVersions.Has "rbac.authorization.k8s.io/v1beta1" -}}
rbac.authorization.k8s.io/v1beta1
{{- else if .Capabilities.APIVersions.Has "rbac.authorization.k8s.io/v1alpha1" -}}
rbac.authorization.k8s.io/v1alpha1
{{- else -}}
rbac.authorization.k8s.io/v1
{{- end -}}
{{- end -}}
{{- define "APIVersion" -}}
{{- if .Capabilities.APIVersions.Has "apps/v1" -}}
apps/v1
{{- else if .Capabilities.APIVersions.Has "extensions/v1beta1" -}}
extensions/v1beta1
{{- else -}}
apps/v1
{{- end -}}
{{- end -}}



{{- define "node.kubeletRootPath" }}
{{- if eq .Values.nodeProfileName "windows" }}
{{- .Values.nodeProfile.kubeletRootPath | default "C:/var/lib/kubelet" -}}
{{- else }}
{{- .Values.nodeProfile.kubeletRootPath | default "/var/lib/kubelet" -}}
{{- end }}
{{- end }}



{{- define "node.fluentConfMountPath" }}
{{- if eq .Values.nodeProfileName "windows" }}
{{- .Values.nodeProfile.fluentConfMountPath | default "C:/etc/fluent-conf" -}}
{{- else }}
{{- .Values.nodeProfile.fluentConfMountPath | default "/fluentd/etc" -}}
{{- end }}
{{- end }}


{{- define "node.volumes" }}
- name: fluentconf
  emptyDir: {}
{{- if eq .Values.nodeProfileName "windows" }}
- name: var
  hostPath:
   path: {{ .Values.nodeProfile.varHostPath | default "C:/var" }}
- name: varlibdocker
  hostPath:
   path: {{ .Values.nodeProfile.varLibDockerContainersHostPath | default "C:/ProgramData/docker" }}
{{- else }}
- name: kubeletroot
  hostPath:
   path: {{ .Values.nodeProfile.kubeletRootHostPath | default "/var/lib/kubelet" }}
- name: varlog
  hostPath:
   path: {{ .Values.nodeProfile.varLogHostPath | default "/var/log" }}
- name: varlibdockercontainers
  hostPath:
   path: {{ .Values.nodeProfile.varLibDockerContainersHostPath | default "/var/lib/docker/containers" }}
{{- end }}
{{- end }}


{{- define "node.fluendVolumeMounts" }}
- name: fluentconf
  mountPath: {{ include "node.fluentConfMountPath" . }}
{{- if eq .Values.nodeProfileName "windows" }}
- name: var
  mountPath: {{ .Values.nodeProfile.varMountPath | default "C:/var" }} 
- name: varlibdocker
  mountPath: {{ .Values.nodeProfile.varLibDockerContainersMountPath | default "C:/ProgramData/docker" }}
  readOnly: true
{{- else }}
- name: kubeletroot
  mountPath: {{ include "node.kubeletRootPath" . }}
  readOnly: true
- name: varlog
  mountPath: {{ .Values.nodeProfile.varLogMountPath | default "/var/log" }}
- name: varlibdockercontainers
  mountPath: {{ .Values.nodeProfile.varLibDockerContainersMountPath | default "/var/lib/docker/containers" }}
  readOnly: true
{{- end }}
{{- end }}


{{- define "node.operatorVolumeMounts" }}
- name: fluentconf
{{- if eq .Values.nodeProfileName "windows" }}
  mountPath: {{ .Values.nodeProfile.fluentConfMountPath | default "C:/etc/fluent-conf" }}
{{- else }}
  mountPath: {{ .Values.nodeProfile.fluentConfMountPath | default "/fluentd/etc" }}
{{- end }}
{{- end }}


{{- define "node.operatorBinary" }}
{{- if eq .Values.nodeProfileName "windows" }}
{{- .Values.nodeProfile.operatorBinary | default "c:/bin/config-reloader.exe" -}}
{{- else }}
{{- .Values.nodeProfile.operatorBinary | default "/bin/config-reloader" -}}
{{- end }}
{{- end }}


{{- define "node.fluentdBinary" }}
{{- if eq .Values.nodeProfileName "windows" }}
{{- .Values.nodeProfile.fluentdBinary | default "fluentd -p /etc/fluent-plugin" -}}
{{- else }}
{{- .Values.nodeProfile.fluentdBinary | default "/usr/local/bundle/bin/fluentd -p /fluentd/plugins" -}}
{{- end }}
{{- end }}

{{- define "node.templatesDir" }}
{{- if eq .Values.nodeProfileName "windows" }}
{{- .Values.nodeProfile.templatesDir | default "C:/templates" -}}
{{- else }}
{{- .Values.nodeProfile.templatesDir | default "/templates" -}}
{{- end }}
{{- end }}

{{- define "node.useSystemd" }}
{{- if .Values.nodeProfile.useSystemd }}
- {{ .Values.nodeProfile.useSystemd }}
{{- else }}
{{- if ne .Values.nodeProfileName "windows" }}
- {{ "--use-systemd" }}
{{- end }}
{{- end }}
{{- end }}