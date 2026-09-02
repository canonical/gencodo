.. _ref_{{ .Root }}_cli:

{{ .Root }} (CLI)
{{ repeat "=" (len .Root) }}======

The :program:`{{ .Root }}` utility exposes the following commands,
each with its own set of options,
and also has a number of global flags:

-h, --help

   Print the help message for the command.

.. rubric:: Commands
{{ range .Commands }}
- :ref:`{{ .Ref }}`: {{ .Short }}
{{- end }}

{{ range .Files }}
.. include:: {{ . }}
{{ end }}
