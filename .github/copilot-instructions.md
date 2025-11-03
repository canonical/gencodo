# GitHub Copilot Instructions for Gencodo

## Project Overview

Gencodo is a **Go library** (not an application) that generates format-agnostic documentation from Cobra CLI applications using Go templates. It's distributed as `github.com/canonical/gencodo` and consumed by other projects via `go get`.

**Key architectural principle**: Flexibility through Go templates - users provide their own templates (Markdown, reST, JSON, etc.), Gencodo extracts structured data from Cobra commands.

## Code Structure & Core Components

### Main Public API (`gencodo.go`)
- **`GenDocs()`**: Generates documentation for a single command to an `io.Writer`
- **`GenDocsTree()`**: Recursively generates docs for all subcommands, creates index file
- **`ExampleParser`**: Configurable parser that extracts structured examples from Cobra's example strings
  - Splits by `\n\n` (double newlines) into separate examples
  - Detects command lines via prefixes (`$`, `>`, `#`) or indentation (≥2 spaces by default)
  - Returns `[]ExampleInfo` with separate `Info` (description) and `Usage` (command) fields

### Data Structures Exposed to Templates
```go
// Available in SingleCommandTemplate:
.Ref             // e.g., "ref_myapp_subcommand" (spaces → underscores)
.CommandName     // Full command path, e.g., "myapp subcommand"
.Short           // Short description
.Long            // Long description (falls back to .Short if empty)
.Synopsis        // Command usage line from Cobra
.Examples        // []ExampleInfo (structured, parsed examples)
.Flags           // []FlagInfo (non-inherited flags only)
.HeadingLen      // Length of CommandName (for formatting)
.RelatedCommands // []string from Annotations["related"] (comma-separated)

// Available in IndexTemplate:
.Files           // []string (generated filenames, sorted alphabetically)
```

### Template Functions
- `indent <spaces> <strings...>`: Indents multi-line strings
- `repeat <count> <string>`: Repeats a string N times
- `replaceSpaces <string>`: Replaces spaces with underscores (used for filenames)

## Development Workflow

### Building & Testing
```bash
make              # Default: fmt + vet + test
make test         # Run all tests
make test-coverage # Generate coverage.html report
make fmt          # Format code (required before commits)
make vet          # Static analysis
```

**Important**: This is a library - `make build` only verifies compilation, there's no executable.

### Testing Patterns (from `gencodo_test.go`)
1. **Use `t.TempDir()`** for file operations - automatic cleanup
2. **Helper functions**: `fileExists()`, `readDirNames()`, `containsAll()`
3. **Template testing approach**: Define inline template with field accessors like `{{ .CommandName }}|{{ .Short }}`
4. **Integration tests**: Use actual template files from `examples/` (see `TestReStructuredTextTemplates`, `TestMarkdownTemplates`)
5. **Example parser tests**: Each test creates `ExampleParser` with specific config, validates `Info`/`Usage` splitting

### Example Template Structure
- **`examples/cli.rst`**: Index template (lists all command files)
- **`examples/command.rst`**: Single command template (reStructuredText format)
- **`examples/cli.md`** & **`examples/command.md`**: Markdown equivalents
- Templates demonstrate conditional sections: `{{- if .Examples }}...{{- end }}`

## Cobra Integration Specifics

### Flag Handling
- Only **non-inherited flags** are extracted (use `cmd.NonInheritedFlags()`)
- Persistent flags from parent commands are intentionally excluded
- Test this with nested commands: `TestGenDocsNonInheritedFlags`

### Command Filtering
- `GenDocsTree()` skips commands where `!c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand()`
- Root command is never documented directly, only its subcommands
- Filenames: `<commandpath-with-dashes>.rst` (e.g., `myapp-sub-cmd.rst`)

### Annotations Convention
```go
cmd.Annotations = map[string]string{
    "related": "foo bar,bar,baz", // Comma-separated, trimmed automatically
}
```

## Common Patterns

### Adding New Template Data
1. Add field to anonymous struct in `GenDocs()` data initialization
2. Update test cases to verify the field works in templates
3. Document in `README.md` under "Cobra Command Structure"

### Testing New Example Parser Behavior
- Create `ExampleParser` with specific `CommandPrefixes` and `MinIndent`
- Test both "happy path" (examples split correctly) and fallback (no prefix/indent match → full text as Usage)
- Validate multiline scenarios (`\n` preserved in `Usage` field)

## License & Headers

**LGPL-3.0-only** - All Go files must include the 18-line SPDX header (see top of `gencodo.go`). Copyright holder: `Canonical Ltd.`

## When Adding Features

1. **Preserve backward compatibility**: This is a library - breaking API changes impact consumers
2. **Template flexibility first**: If a feature could be solved with templates + new data fields, prefer that over code logic
3. **Test with both formats**: Validate changes work with both `.rst` and `.md` example templates
4. **Update README.md**: Document new fields/functions in "Templates" or "Helper functions" sections
