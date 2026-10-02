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
{{- define "gobetween.sccName" -}}
{{- if .Values.openshift.sccName -}}
{{- .Values.openshift.sccName -}}
{{- else if .Values.openshift.createSCC -}}
{{- $name := printf "%s-%s-privileged" .Release.Namespace (include "gobetween.fullname" .) -}}
{{- if gt (len $name) 63 -}}
{{- printf "%s-%s" ($name | trunc 54 | trimSuffix "-") ($name | sha256sum | trunc 8) -}}
{{- else -}}
{{- $name -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- define "gobetween.podSecurityContext" -}}
{{- $context := deepCopy .Values.podSecurityContext -}}
{{- if .Values.openshift.privileged -}}
{{- $_ := set $context "runAsUser" 0 -}}
{{- $_ := set $context "runAsNonRoot" false -}}
{{- $_ := unset $context "seccompProfile" -}}
{{- end -}}
{{- toYaml $context -}}
{{- end -}}
{{- define "gobetween.containerSecurityContext" -}}
{{- $context := deepCopy .Values.containerSecurityContext -}}
{{- if .Values.openshift.privileged -}}
{{- $_ := set $context "privileged" true -}}
{{- $_ := set $context "allowPrivilegeEscalation" true -}}
{{- $_ := set $context "runAsUser" 0 -}}
{{- $_ := set $context "runAsNonRoot" false -}}
{{- $_ := unset $context "seccompProfile" -}}
{{- end -}}
{{- toYaml $context -}}
{{- end -}}
{{- define "gobetween.validate" -}}
{{- if and .Values.openshift.createSCC (not .Values.openshift.privileged) -}}
{{- fail "openshift.createSCC requires openshift.privileged=true; the managed SCC is a privileged profile" -}}
{{- end -}}
{{- if and .Values.openshift.privileged (not (include "gobetween.sccName" .)) -}}
{{- fail "privileged mode requires openshift.createSCC=true or an approved openshift.sccName" -}}
{{- end -}}
{{- if and .Values.openshift.privileged (not .Values.hostUsers) -}}
{{- fail "the root/privileged profile requires hostUsers=true" -}}
{{- end -}}
{{- if and (not .Values.openshift.privileged) .Values.containerSecurityContext.privileged -}}
{{- fail "use openshift.privileged=true for coherent root/privileged security contexts" -}}
{{- end -}}
{{- if and .Values.openshift.createSCC (has .Values.openshift.sccName (list "privileged" "restricted" "restricted-v2" "restricted-v3" "anyuid" "nonroot" "nonroot-v2" "hostaccess" "hostmount-anyuid" "hostnetwork" "hostnetwork-v2" "node-exporter" "nested-container")) -}}
{{- fail "do not create/overwrite a built-in SCC; use createSCC=false to reference an existing approved SCC" -}}
{{- end -}}
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
