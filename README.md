<!-- SPDX-License-Identifier: LGPL-3.0-only -->
<!-- Copyright 2025-2026 Canonical Ltd. -->

# Gencodo

Gencodo is a **template-driven** documentation generator for [Cobra](https://github.com/spf13/cobra) CLI applications.

While Cobra provides built-in documentation generators for Markdown, reStructuredText, and man pages, these use hardcoded formats that may not match your project's documentation style or toolchain requirements. **Gencodo solves this by letting you provide your own [Go templates](https://pkg.go.dev/text/template)**, giving you complete control over the output format, whether that's custom Markdown flavors, reStructuredText with specific directives, JSON/YAML schemas, or any other text-based format.

Flexibility through templates is the key reason for Gencodo to exist; it extracts structured data from your Cobra commands (metadata, flags, parsed examples, relationships) and passes it to your templates, allowing you to format documentation exactly how you need it.

## Why Gencodo?

Use Cobra's built-in generators when:
- The default Markdown/reST/man page formats work for you
- You don't need custom formatting or documentation structure

Use Gencodo when:
- You need custom documentation formats (specific Markdown/reST variants, JSON, YAML, etc.)
- Your documentation toolchain requires specific markup patterns or directives
- You want structured example parsing (separate descriptions from commands)
- You need fine-grained control over how commands, flags, and examples are presented
- You're integrating with existing documentation that has established formatting conventions

See the `examples/` directory for reference templates in Markdown and reStructuredText formats.

## Usage

```shell
go get github.com/canonical/gencodo@latest
```

In your code:

```go
import "github.com/canonical/gencodo"

templates := gencodo.TemplateInfo{
    IndexFileName:         "cli.rst",
    IndexTemplate:         indexTemplate,   // string
    SingleCommandTemplate: commandTemplate, // string
}

// reStructuredText: one .rst file per command plus the index
err := gencodo.GenRSTTree(rootCmd, "docs/reference", templates, nil)

// Markdown: one .md file per command plus the index
err = gencodo.GenMarkdownTree(rootCmd, "docs/reference", templates, nil)

// A single command to any io.Writer, in any format
err = gencodo.GenDocs(cmd, os.Stdout, commandTemplate)
```

- `rootCmd`: your Cobra root command. The tree generators document every available subcommand (recursively) and skip the root itself, hidden and deprecated commands, and help topics.
- `"docs/reference"`: output directory, created if missing. Command files are named after the command path with dashes (`myapp-group-list.rst`); the index file is named by `IndexFileName`.
- `templates`: a `TemplateInfo` with your index and command templates.
- The fourth argument is an optional *file prepender*: a `func(path string) string` whose result is written at the top of every command file (for example a Hugo front matter block). Pass `nil` if you don't need one.
- Trailing `...Option` values tune generation; see [Options](#options).

One way to ship templates with your binary is [embedding](https://pkg.go.dev/embed):

```go
//go:embed gendocs/cli.rst gendocs/command.rst
var templates embed.FS
```

For more usage examples, see the [test code](https://github.com/canonical/gencodo/blob/main/gencodo_test.go).

### Options

All generators accept functional options:

| Option | Effect |
| --- | --- |
| `WithExampleParser(parser)` | Use a custom [`ExampleParser`](#examples) instead of the default prefixes/indentation rules. |
| `WithFuncs(template.FuncMap)` | Add functions to both templates. Names that clash with [built-in helpers](#helper-functions) override them. |
| `WithDryRun()` | Render everything but write nothing (tree generators only). Handy to validate templates against the real command tree in CI. |
| `WithHiddenFlags()` | Include hidden flags (and deprecated flags, which pflag hides) in `.Flags` and `.InheritedFlags`. |
| `WithoutHelpFlag()` | Omit the `--help` flag that Cobra adds automatically. Help flags your application defines itself are kept. |

```go
err := gencodo.GenRSTTree(rootCmd, dir, templates, nil,
    gencodo.WithExampleParser(gencodo.ExampleParser{CommandPrefixes: []string{"$"}, DisableIndentDetection: true}),
    gencodo.WithFuncs(template.FuncMap{"code": func(s string) string { return "``" + s + "``" }}),
)
```

Adding the variadic options is source-compatible for call sites; only code that stored a generator in a variable of the old function type needs updating.

### Validating templates

`ValidateTemplates` parses both templates and renders them against synthetic data (a fully populated command, a minimal one, and an index) without touching the file system. It catches unknown fields, unknown functions, and errors in `{{else}}` branches. Run it in a unit test so that a template change can't break documentation generation later:

```go
func TestDocTemplates(t *testing.T) {
    if err := gencodo.ValidateTemplates(templates); err != nil {
        t.Fatal(err)
    }
}
```

Pass the same options you use for generation, so functions added with `WithFuncs` are known. For a check against your actual commands, run a tree generator with `WithDryRun()`.

## Templates

Gencodo uses [Go templates](https://pkg.go.dev/text/template), giving you full access to Go's templating capabilities. There's no restriction on output format - generate Markdown, reStructuredText, YAML, JSON, or any text-based format your documentation workflow requires. Templates are executed with `missingkey=error`, so a typo in a field name fails generation instead of rendering an empty string. For map lookups that may be absent (such as `.Annotations`), use `{{ index .Annotations "key" }}`.

**Example templates**: the `examples/` directory contains reference implementations for both Markdown and reStructuredText:
- `examples/command.md` / `examples/command.rst` - individual command documentation templates
- `examples/cli.md` / `examples/cli.rst` - index/table of contents templates

They demonstrate conditional sections, flag iteration with shorthands and defaults, structured examples, subcommand listings, and cross-links.

Gencodo requires two templates via the `TemplateInfo` struct:

1. `SingleCommandTemplate`: rendered once per command by `GenDocs` and the tree generators. Receives a `CommandData` value.
2. `IndexTemplate`: rendered once per tree by the tree generators, after all command files. Receives an `IndexData` value. The output file name is `IndexFileName`.

### Command template data (`CommandData`)

| Field | Type | Description |
| --- | --- | --- |
| `.CommandName` | string | Full command path, e.g. `myapp group list` |
| `.Ref` | string | Reference ID: `ref_` + command path with underscores, e.g. `ref_myapp_group_list` |
| `.File` | string | File name of this command's document, e.g. `myapp-group-list.rst` (no extension with `GenDocs`) |
| `.Short` | string | Short description |
| `.Long` | string | Long description; falls back to `.Short` if empty |
| `.Synopsis` | string | Cobra usage line, e.g. `myapp group list [NAME] [flags]` |
| `.HeadingLen` | int | Length of `.CommandName` in characters, for heading underlines |
| `.Examples` | []ExampleInfo | Parsed examples, each with `.Info` (description) and `.Usage` (command lines, verbatim) |
| `.Flags` | []FlagInfo | Flags defined on this command (see below) |
| `.InheritedFlags` | []FlagInfo | Persistent flags inherited from parent commands |
| `.Subcommands` | []CommandSummary | Available subcommands: `.Name`, `.Short`, `.Ref`, `.File` |
| `.Parent` | string | Parent command path, or empty for the root |
| `.Aliases` | []string | Command aliases |
| `.Deprecated` | string | Deprecation message, or empty |
| `.Runnable` | bool | Whether the command has a `Run` function |
| `.RelatedCommands` | []string | From `Annotations["related"]`, comma-separated command paths |
| `.Annotations` | map[string]string | Raw command annotations |

`FlagInfo` fields: `.Name`, `.Shorthand`, `.Usage`, `.DefaultValue`, `.Type` (pflag type name such as `string`, `bool`, `stringSlice`), `.NoOptDefVal`, `.Deprecated`, `.ShorthandDeprecated`, `.Hidden`, `.Required`, `.SetByCobra` (true for the automatic `--help`).

Hidden flags are omitted by default, as in Cobra's help output. Note that pflag's `MarkDeprecated` also hides a flag, so deprecated flags only appear with `WithHiddenFlags()`.

### Index template data (`IndexData`)

| Field | Type | Description |
| --- | --- | --- |
| `.Root` | string | Root command name |
| `.Files` | []string | Generated command file names, in generation order |
| `.Commands` | []CommandSummary | Generated commands (`.Name`, `.Short`, `.Ref`, `.File`), in the same order |

Order follows Cobra's `Commands()` (alphabetical unless `cobra.EnableCommandSorting` is off), depth-first.

### Helper functions

Available in both templates. String arguments come last so the helpers compose in pipelines.

| Helper | Example | Result |
| --- | --- | --- |
| `indent N s...` | `{{ .Usage \| indent 3 }}` | Indents every line of each string by N spaces |
| `repeat s N` | `{{ repeat "-" .HeadingLen }}` | Repeats a string |
| `replaceSpaces s` | `{{ "a b" \| replaceSpaces }}` | `a_b` |
| `slug s` | `{{ "My App sub!" \| slug }}` | `my-app-sub` (lowercase, non-alphanumerics to dashes, trimmed) |
| `anchor s` | `{{ "ref_my app" \| anchor }}` | `ref_my-app` (like `slug`, keeps underscores) |
| `titleCase s` | `{{ "list all" \| titleCase }}` | `List All` |
| `trimPrefix p s` | `{{ .CommandName \| trimPrefix "myapp " }}` | `group list` |
| `trimSuffix p s` | `{{ .Short \| trimSuffix "." }}.` | Exactly one trailing period |
| `trimSpace s` | `{{ .Long \| trimSpace }}` | Drops leading/trailing blank lines |
| `lower s`, `upper s` | `{{ .Name \| upper }}` | Case conversion |
| `join sep list` | `{{ .Aliases \| join ", " }}` | `g, grp` |
| `replace old new s` | `{{ . \| replace " " "-" }}` | Replaces all occurrences |

Add your own with `WithFuncs`.

### Link formatting patterns

Since 0.2.0 there is no link handler callback; links are produced entirely in templates from the fields above. Some patterns:

**reStructuredText / Sphinx** - every command file starts with a label built from `.Ref`, so any page can reference it with `:ref:`:

```rst
.. _{{ .Ref }}:

{{ range .Subcommands }}
- :ref:`{{ .Ref }}`
{{- end }}
{{ range .RelatedCommands }}
- :ref:`ref_{{ . | replaceSpaces }}`
{{- end }}
```

**Markdown** - link to sibling files by `.File`, or to headings by anchor:

```markdown
{{ range .Subcommands }}
- [{{ .Name }}]({{ .File }}): {{ .Short }}
{{- end }}
- [{{ .CommandName }}](#{{ .CommandName | slug }})
```

**Hugo / front matter** - use the file prepender for metadata that needs the file path, and `.File`/`slug` for permalinks:

```go
prepender := func(path string) string {
    name := strings.TrimSuffix(filepath.Base(path), ".md")
    return fmt.Sprintf("---\ntitle: %q\nslug: %s\n---\n\n", name, name)
}
err := gencodo.GenMarkdownTree(rootCmd, dir, templates, prepender)
```

**Index / table of contents** - `.Commands` carries names, short descriptions, refs, and file names, so an index needs no extra lookups:

```rst
{{ range .Commands }}
- :ref:`{{ .Ref }}`: {{ .Short }}
{{- end }}
```

## Cobra command structure

Gencodo extracts and transforms Cobra command metadata as follows.

Used unchanged: `Use`, `Short`, `Long` (falls back to `Short`), `Aliases`, `Deprecated`, `Annotations`, the command hierarchy.

Transformed:
- **Examples** are parsed into `ExampleInfo` blocks with separate `Info` (description) and `Usage` (command) fields. The example string is split into blocks by a blank line (`BlockSeparator`, default `"\n\n"`), and the first command line of each block is detected by:
  - Command prefixes: `$`, `>`, `#` (configurable via `CommandPrefixes`)
  - Indentation: lines indented 2+ spaces or a tab (configurable via `MinIndent`; disable with `DisableIndentDetection`)

  Everything before the first command line is `Info`; the command line and the rest of the block is `Usage`, kept verbatim (prefix and indentation included). Note that a description line starting with `#` or indented by two spaces counts as a command under the defaults; adjust the parser with `WithExampleParser` if your examples use such lines.
- **Flags** become `FlagInfo` values. `.Flags` holds the command's own flags, `.InheritedFlags` the persistent flags of its parents.

Behaviour notes:
- Like Cobra's own generators, gencodo calls `InitDefaultHelpCmd` and `InitDefaultHelpFlag` on each command, so `--help` appears in `.Flags` (mark: `.SetByCobra`) and `[flags]` in `.Synopsis`. Use `WithoutHelpFlag()` to drop the flag. This mutates the command tree; don't share one tree between goroutines.
- The tree generators skip the root command, hidden and deprecated commands, the `help` command, and help topics (commands with neither `Run` nor subcommands). `.Subcommands` applies the same filter.
- Command names containing dashes can collide with file names of nested commands (`foo-bar` vs `foo bar`), as with Cobra's generators.
- Files are rendered fully in memory and written only on success, so a template error never leaves a partial file behind.

## Development

```shell
make install        # golangci-lint and reuse (via pipx)
make                # fmt, vet, lint, test
make test-race      # tests with the race detector
make test-coverage  # coverage summary and coverage.html
make reuse          # REUSE license/copyright compliance
```

CI runs the same checks on the minimum supported Go version (`go.mod`), the previous stable, and the current stable release, and publishes the coverage profile as a workflow artifact.

## Release process (maintainers)

1. Move the `[Unreleased]` entries in `CHANGELOG.md` into a new `## [X.Y.Z] - YYYY-MM-DD` section (note breaking changes and migrations) and update the compare links at the bottom. Update README if the API changed.
2. Commit, then run `make release VERSION=vX.Y.Z`. It checks the CHANGELOG section exists, runs vet and the race tests, tags, and pushes the tag.
3. The `Release` workflow publishes the GitHub release with that CHANGELOG section as its notes; check `gh release view vX.Y.Z`.
4. Optionally run `make proxy-warm VERSION=vX.Y.Z` so `proxy.golang.org` indexes the version immediately.

## License and copyright

Gencodo is licensed under the [LGPL-3.0-only](LICENSES/LGPL-3.0-only.txt) and follows the [REUSE](https://reuse.software) specification: every file carries `SPDX-License-Identifier` and `Copyright` information, either in a header or through `REUSE.toml` (for files that cannot hold a comment, such as the example templates). `make reuse` and the CI `reuse` job verify this.

## Contributing

Contributions are welcome! Please submit a PR or open an issue for discussions. Run `make` before pushing; CI enforces formatting, vet, lint, tests, and REUSE compliance.
