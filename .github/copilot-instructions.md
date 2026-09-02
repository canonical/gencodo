<!-- SPDX-License-Identifier: LGPL-3.0-only -->
<!-- Copyright 2025-2026 Canonical Ltd. -->

# GitHub Copilot Instructions for Gencodo

## Project Overview

Gencodo is a **Go library** (not an application) that generates format-agnostic documentation from Cobra CLI applications using Go templates. It's distributed as `github.com/canonical/gencodo` and consumed by other projects via `go get`.

**Key architectural principle**: Flexibility through Go templates - users provide their own templates (Markdown, reST, JSON, etc.), Gencodo extracts structured data from Cobra commands. Prefer exposing data to templates over adding logic to the library.

## Copilot Agents

Role-based agents are available in `.github/agents/` for specialized guidance:

- **`design.agent.md`**: Product strategy, user value, architecture decisions, and system design
- **`engineering.agent.md`**: Code implementation, technical leadership, and development best practices
- **`cicd.agent.md`**: CI/CD pipelines, testing strategies, and quality assurance

Invoke agents by referencing them in your prompts (e.g., "@design help me prioritize features" or "@engineering review this code").

## Code Structure & Core Components

### Main Public API (`gencodo.go`)
- **`GenDocs(cmd, w, template, opts...)`**: Generates documentation for a single command to an `io.Writer`
- **`GenMarkdownTree` / `GenRSTTree(cmd, dir, templates, filePrepender, opts...)`**: Recursively generate docs for all subcommands (skipping the root, hidden/deprecated commands, and help topics) and create an index file
- **`ValidateTemplates(templates, opts...)`**: Renders both templates against synthetic data without writing files
- **Options**: `WithExampleParser`, `WithFuncs`, `WithDryRun`, `WithHiddenFlags`, `WithoutHelpFlag` (type `Option func(*config)`)
- **`ExampleParser`**: Configurable parser that extracts structured examples from Cobra's example strings
  - Splits by `BlockSeparator` (default `\n\n`) into separate examples
  - Detects command lines via `CommandPrefixes` (`$`, `>`, `#`) or indentation (`MinIndent`, default 2; `DisableIndentDetection` turns it off)
  - Returns `[]ExampleInfo` with separate `Info` (description) and `Usage` (command, verbatim) fields
- Internals: `config` (options), `commandData` (extraction), `genDocs` (render to buffer), `genDocsTree` (walk + write), `builtinFuncs` (template helpers). Files are rendered in memory and written with `os.WriteFile` only on success.

### Data Structures Exposed to Templates
- `CommandData` (SingleCommandTemplate): `.Ref`, `.CommandName`, `.File`, `.Short`, `.Long`, `.Synopsis`, `.HeadingLen`, `.Examples`, `.Flags`, `.InheritedFlags`, `.Subcommands`, `.Parent`, `.Aliases`, `.Deprecated`, `.Runnable`, `.RelatedCommands`, `.Annotations`
- `FlagInfo`: `.Name`, `.Shorthand`, `.Usage`, `.DefaultValue`, `.Type`, `.NoOptDefVal`, `.Deprecated`, `.ShorthandDeprecated`, `.Hidden`, `.Required`, `.SetByCobra`
- `IndexData` (IndexTemplate): `.Root`, `.Files`, `.Commands` (each `CommandSummary`: `.Name`, `.Short`, `.Ref`, `.File`)
- Order of files/commands follows Cobra's `Commands()` order, depth-first (not alphabetically sorted by gencodo)

### Template Functions
Built-ins (string argument last, pipeline-friendly): `indent`, `repeat`, `replaceSpaces`, `slug`, `anchor`, `titleCase`, `trimPrefix`, `trimSuffix`, `trimSpace`, `lower`, `upper`, `join`, `replace`. Users add more with `WithFuncs`. Both templates get the same function map and run with `missingkey=error`.

## Development Workflow

- `make` runs fmt, vet, lint (golangci-lint v2, `.golangci.yaml`), and tests; `make test-race`, `make test-coverage`, `make reuse` are also available
- Tests live in `gencodo_test.go` (single package, inline templates, `t.TempDir()`); the reference templates in `examples/` must keep passing `ValidateTemplates`
- Every behaviour change needs a test and a CHANGELOG entry under `[Unreleased]`
- Releases: update CHANGELOG, `make release VERSION=vX.Y.Z`; the `Release` workflow publishes the GitHub release from the CHANGELOG section

## License & Copyright

**LGPL-3.0-only**, REUSE-compliant. Copyright holder: `Canonical Ltd.`
- Go files: the full LGPL notice header (see top of `gencodo.go`), which includes the SPDX license identifier line
- All other files: a two-line header with the SPDX license identifier and `Copyright <year> Canonical Ltd.` in the file's comment syntax (`#` for Makefile/YAML, `<!-- -->` for Markdown); copy it from `Makefile` or `README.md`
- Files that cannot carry a header (`examples/` templates, `go.sum`, `LICENSE`) are covered by `REUSE.toml`
- `make reuse` / the CI `reuse` job must pass
