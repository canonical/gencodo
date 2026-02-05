# Gencodo

Gencodo is a **template-driven** documentation generator for [Cobra](https://github.com/spf13/cobra) CLI applications.

While Cobra provides built-in documentation generators for Markdown, reStructuredText, and man pages, these use hardcoded formats that may not match your project's documentation style or toolchain requirements. **Gencodo solves this by letting you provide your own [Go templates](https://pkg.go.dev/text/template)**, giving you complete control over the output format, whether that's custom Markdown flavors, reStructuredText with specific directives, JSON/YAML schemas, or any other text-based format.

Flexibility through templates is the key reason for Gencodo to exist; it extracts structured data from your Cobra commands (metadata, flags, parsed examples) and passes it to your templates, allowing you to format documentation exactly how you need it.

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

To generate documentation for a CLI application:

```shell
go get github.com/canonical/gencodo@latest
```

In your code:

```go
import (
    "github.com/canonical/gencodo"
)

// For Markdown documentation
gencodo.GenMarkdownTree(rootCmd, "docs", templates, filePrepender)

// For reStructuredText documentation
gencodo.GenRSTTree(rootCmd, "docs", templates, filePrepender)
```

- `rootCmd`: Your Cobra root command.
- `docs`: Output directory for documentation files.
- `templates`: `TemplateInfo` struct containing your custom templates for index and command files.
- `filePrepender`: Function to prepend headers to files.

These arguments are similar to [Cobra's own implementation](https://umarcor.github.io/cobra/#generating-restructured-text-docs-for-your-own-cobracommand);
the main addition is `templates`, which provides custom templates for documentation formatting.

One way to add them to your code is [embedding](https://pkg.go.dev/embed), for instance:

```go
//go:embed gendocs/cli.rst gendocs/command.rst
var templates embed.FS
```

For more usage examples, see the [test code](https://github.com/canonical/gencodo/blob/main/gencodo_test.go).

### Cobra Command Structure

Gencodo extracts and transforms Cobra command metadata as follows:

Used unchanged:
- `Use`: Command name and syntax
- `Short`: Short description
- `Long`: Detailed description (falls back to `Short` if empty)
- Command hierarchy and related commands

Transformed:
- Examples: Parsed into structured `ExampleInfo` blocks containing separate `Info` (description) and `Usage` (command) fields. Examples are split by double newlines (`\n\n`), and command lines are detected by:
  - Command prefixes: `$`, `>`, `#` (configurable via `ExampleParser`)
  - Indentation: Lines with 2+ spaces (configurable via `MinIndent`)
- Flags: Extracted into `FlagInfo` structs with `Name`, `Usage`, and `DefaultValue` fields (non-inherited flags only)

Helper functions:
- `replaceSpaces`: Replaces spaces with a specified character (useful for filenames)
- `headingLen`: Returns the length of a command name (for heading formatting)

These extracted fields are available in your templates for flexible formatting.

## Templates

Gencodo uses [Go templates](https://pkg.go.dev/text/template), giving you full access to Go's templating capabilities. There's no restriction on output format - generate Markdown, reStructuredText, YAML, JSON, or any text-based format your documentation workflow requires.

**Example templates**: The `examples/` directory contains reference implementations for both Markdown and reStructuredText:
- `examples/command.md` / `examples/command.rst` - Individual command documentation templates
- `examples/cli.md` / `examples/cli.rst` - Index/table of contents templates

These demonstrate common patterns like conditional sections, flag iteration, and structured example formatting.

Gencodo requires two template types via the `TemplateInfo` struct:

1. `SingleCommandTemplate`: Template for individual command documentation files. Used by `GenDocs()` to generate one file per command, containing command metadata like name, description, synopsis, flags, examples, and related commands. Each file is named after the command path (e.g., `my-app-subcommand.rst`).

2. `IndexTemplate`: Template for the index/table of contents file. Used by `GenDocsTree()` to generate a single file listing all generated command documentation files. Receives a `Files` array containing all generated filenames. The output filename is set via `IndexFileName` in the `TemplateInfo` struct.

## Contributing

Contributions are welcome! Please submit a PR or open an issue for discussions.

## License

[LGPLv3](https://www.gnu.org/licenses/lgpl-3.0.en.html#license-text)
