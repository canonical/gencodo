.. _{{ .Ref }}:

{{ .CommandName }}
{{ repeat "-" .HeadingLen }}

.. @artefact {{ .CommandName }}

{{ .Short | trimSuffix "." }}.

{{- if .Deprecated }}

.. deprecated:: {{ .Deprecated }}
{{- end }}

.. rubric:: Usage

.. code-block:: console

   $ {{ .Synopsis }}

.. rubric:: Description

{{ .Long | trimSpace }}

{{- if .Examples }}

.. rubric:: Examples
{{ range .Examples }}
{{ .Info }}

.. code-block:: console

{{ .Usage | indent 3 }}
{{ end }}
{{- end }}

{{- if .Flags }}

.. rubric:: Flags
{{ range .Flags }}
{{ if .Shorthand }}-{{ .Shorthand }}, {{ end }}--{{ .Name }}{{ if .Required }} (required){{ end }}

{{ .Usage | indent 3 }}
{{- if and .DefaultValue (ne .Type "bool") }}

   Default: ``{{ .DefaultValue }}``
{{- end }}
{{ end }}
{{- end }}

{{- if .Subcommands }}

.. rubric:: Subcommands
{{ range .Subcommands }}
- :ref:`{{ .Ref }}`
{{- end }}
{{- end }}

{{- if .RelatedCommands }}

.. rubric:: See also
{{ range .RelatedCommands }}
- :ref:`ref_{{ . | replaceSpaces }}`
{{- end }}
{{- end }}
