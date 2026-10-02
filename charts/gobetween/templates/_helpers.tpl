{{- define "gobetween.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "gobetween.fullname" -}}
{{- default (printf "%s-%s" .Release.Name (include "gobetween.name" .)) .Values.fullnameOverride | trunc 54 | trimSuffix "-" -}}
{{- end -}}
{{- define "gobetween.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gobetween.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
{{- define "gobetween.labels" -}}
{{ include "gobetween.selectorLabels" . }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | quote }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}
{{- define "gobetween.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "gobetween.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- required "serviceAccount.name is required when create=false" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}
{{- define "gobetween.needsBPF" -}}
{{- if and (eq .Values.udp.reusePortDistribution "rr") (gt (int .Values.runtime.workerProcesses) 1) -}}true{{- end -}}
{{- end -}}
{{- define "gobetween.validate" -}}
{{- if hasKey .Values.containerSecurityContext "capabilities" -}}
{{- fail "set extraCapabilities instead of containerSecurityContext.capabilities" -}}
{{- end -}}
{{- if lt (int .Values.resources.cpu) (int .Values.runtime.workerProcesses) -}}
{{- fail "resources.cpu must be >= runtime.workerProcesses; physical mode also requires enough COMPLETE physical cores at runtime" -}}
{{- end -}}
{{- if le (int .Values.terminationGracePeriodSeconds) (int .Values.runtime.shutdownTimeoutSeconds) -}}
{{- fail "terminationGracePeriodSeconds must exceed runtime.shutdownTimeoutSeconds" -}}
{{- end -}}
{{- if and (include "gobetween.needsBPF" .) (not .Values.hostUsers) -}}
{{- fail "RR BPF requires hostUsers=true (initial-user-namespace capabilities)" -}}
{{- end -}}
{{- if and (eq .Values.discovery.kind "static") (empty .Values.discovery.staticList) -}}
{{- fail "discovery.staticList must not be empty for static discovery" -}}
{{- end -}}
{{- end -}}
