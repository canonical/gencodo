# {{ .CommandName }}

{{ .Short | trimSuffix "." }}.

{{- if .Deprecated }}

> **Deprecated:** {{ .Deprecated }}
{{- end }}

## Usage

```console
$ {{ .Synopsis }}
```

## Description

{{ .Long | trimSpace }}

{{- if .Examples }}

## Examples
{{ range .Examples }}
{{ .Info }}

```console
{{ .Usage }}
```
{{ end }}
{{- end }}

{{- if .Flags }}

## Flags
{{ range .Flags }}
**{{ if .Shorthand }}-{{ .Shorthand }}, {{ end }}--{{ .Name }}**{{ if .Required }} (required){{ end }}

{{ .Usage | indent 3 }}
{{- if and .DefaultValue (ne .Type "bool") }}

   Default: `{{ .DefaultValue }}`
{{- end }}
{{ end }}
{{- end }}

{{- if .Subcommands }}

## Subcommands
{{ range .Subcommands }}
- [{{ .Name }}]({{ .File }}): {{ .Short }}
{{- end }}
{{- end }}

{{- if .RelatedCommands }}

## See also
{{ range .RelatedCommands }}
- [{{ . }}]({{ . | replace " " "-" }}.md)
{{- end }}
{{- end }}
