# {{ .Root }} (CLI)

The `{{ .Root }}` utility exposes the following commands,
each with its own set of options,
and also has a number of global flags:

**-h, --help**

Print the help message for the command.

## Commands
{{ range .Commands }}
- [{{ .Name }}]({{ .File }}): {{ .Short }}
{{- end }}
