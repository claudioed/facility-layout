{{/*
Expand the name of the chart.
*/}}
{{- define "facility-layout.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "facility-layout.fullname" -}}
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
Chart name and version as used by the chart label.
*/}}
{{- define "facility-layout.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "facility-layout.labels" -}}
helm.sh/chart: {{ include "facility-layout.chart" . }}
{{ include "facility-layout.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "facility-layout.selectorLabels" -}}
app.kubernetes.io/name: {{ include "facility-layout.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "facility-layout.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "facility-layout.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Name of the Secret holding DATABASE_URL, when the chart creates its own.
*/}}
{{- define "facility-layout.databaseSecretName" -}}
{{- if .Values.database.existingSecret }}
{{- .Values.database.existingSecret }}
{{- else }}
{{- include "facility-layout.fullname" . }}-database
{{- end }}
{{- end }}

{{/*
Fully qualified name of the analytics projector deployment (ADR-0010).
*/}}
{{- define "facility-layout.projectorFullname" -}}
{{- include "facility-layout.fullname" . }}-projector
{{- end }}

{{/*
Fully qualified name of the analytics reports deployment/service (ADR-0010).
*/}}
{{- define "facility-layout.reportsFullname" -}}
{{- include "facility-layout.fullname" . }}-reports
{{- end }}

{{/*
Name of the Secret holding the analytics DSNs, when the chart creates its own.
*/}}
{{- define "facility-layout.analyticsSecretName" -}}
{{- if .Values.analytics.database.existingSecret }}
{{- .Values.analytics.database.existingSecret }}
{{- else }}
{{- include "facility-layout.fullname" . }}-analytics
{{- end }}
{{- end }}

{{/*
Fully qualified name of the MCP server deployment/service (ADR-0007).
*/}}
{{- define "facility-layout.mcpFullname" -}}
{{- include "facility-layout.fullname" . }}-mcp
{{- end }}

{{/*
Name of the Secret holding the MCP bearer keys, when the chart creates its own.
*/}}
{{- define "facility-layout.mcpSecretName" -}}
{{- if .Values.mcp.existingSecret }}
{{- .Values.mcp.existingSecret }}
{{- else }}
{{- include "facility-layout.fullname" . }}-mcp
{{- end }}
{{- end }}

{{/*
Name of the Secret holding the REST bearer keys (ADR-0014), when the chart
creates its own.
*/}}
{{- define "facility-layout.authSecretName" -}}
{{- if .Values.auth.existingSecret }}
{{- .Values.auth.existingSecret }}
{{- else }}
{{- include "facility-layout.fullname" . }}-auth
{{- end }}
{{- end }}

{{/*
True when the REST auth Secret refs should be rendered on a pod: either the
chart owns a key or an existing Secret was named.
*/}}
{{- define "facility-layout.authEnabled" -}}
{{- if or .Values.auth.readKey .Values.auth.readWriteKey .Values.auth.existingSecret -}}true{{- end -}}
{{- end }}

{{/*
The REST auth env block shared by the main and reports containers: AUTH_MODE
always (from the ConfigMap), the key refs only when a Secret exists. Each key
is optional so a Secret may carry just one of them.
*/}}
{{- define "facility-layout.authEnv" -}}
- name: AUTH_MODE
  valueFrom:
    configMapKeyRef:
      name: {{ include "facility-layout.fullname" . }}
      key: AUTH_MODE
{{- if include "facility-layout.authEnabled" . }}
- name: API_READ_KEY
  valueFrom:
    secretKeyRef:
      name: {{ include "facility-layout.authSecretName" . }}
      key: API_READ_KEY
      optional: true
- name: API_READWRITE_KEY
  valueFrom:
    secretKeyRef:
      name: {{ include "facility-layout.authSecretName" . }}
      key: API_READWRITE_KEY
      optional: true
{{- end }}
{{- end }}
