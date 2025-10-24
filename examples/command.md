# {{ .CommandName }}

{{ .Short }}.

## Usage

```console
$ {{ .Synopsis }}
```

## Description

{{ .Long }}

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
**--{{ .Name }}**

{{ .Usage | indent 3 }}

{{ end }}
{{- end }}
