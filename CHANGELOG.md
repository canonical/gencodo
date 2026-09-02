<!-- SPDX-License-Identifier: LGPL-3.0-only -->
<!-- Copyright 2025-2026 Canonical Ltd. -->

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While the major version is 0, minor releases may change behaviour; such changes are
called out under **Changed** or marked **BREAKING**.

## [Unreleased]

## [0.3.0] - 2026-09-02

Closes #15, #16, #17, #18, #19, #20.

### Added
- Functional options on `GenDocs`, `GenMarkdownTree`, and `GenRSTTree`: `WithExampleParser`, `WithFuncs`, `WithDryRun`, `WithHiddenFlags`, `WithoutHelpFlag` (#17)
- `ValidateTemplates` renders both templates against synthetic data without writing files, for use in unit tests and CI (#18)
- Built-in template helpers `slug`, `anchor`, `titleCase`, `trimPrefix`, `trimSuffix`, `trimSpace`, `lower`, `upper`, `join`, `replace`; all helpers are now available in the index template too (#16)
- `ExampleParser.BlockSeparator` and `ExampleParser.DisableIndentDetection`; CRLF line endings are normalised and a leading tab counts as indentation (#19)
- Exported `CommandData`, `IndexData`, and `CommandSummary` types describing the template data
- `FlagInfo` gained `Shorthand`, `Type`, `NoOptDefVal`, `Deprecated`, `ShorthandDeprecated`, `Hidden`, `Required`, `SetByCobra`
- Command template data gained `File`, `InheritedFlags`, `Subcommands`, `Parent`, `Aliases`, `Deprecated`, `Runnable`, `Annotations`; index template data gained `Root` and `Commands`
- README: template data reference, helper functions, link formatting patterns, behaviour notes, development and release process (#15)
- `make lint` (golangci-lint v2, config in `.golangci.yaml`), `make reuse`, coverage summary in `make test-coverage`; CI lint, coverage, and REUSE jobs; Dependabot for Go modules and Actions (#20)
- Release workflow publishing a GitHub release from the CHANGELOG section on every `vX.Y.Z` tag; `make release` refuses to tag without that section
- REUSE compliance: `LICENSES/LGPL-3.0-only.txt`, `REUSE.toml`, SPDX headers on all files

### Changed
- Hidden flags (including flags hidden by pflag's `MarkDeprecated`) are omitted by default, matching Cobra's help and doc output; use `WithHiddenFlags()` to include them
- Minimum Go version is 1.23; dependencies bumped to cobra 1.10.2 and pflag 1.0.10
- Output files are rendered in memory and written only on success, so a template error no longer leaves partial files; file close errors are reported
- The command template is parsed once per tree instead of once per command, and template parse errors are reported before any file or directory is created
- `HeadingLen` counts characters rather than bytes
- A `nil` file prepender is accepted
- The generator function types changed (trailing `...Option`); calls are source-compatible, but code that assigned a generator to a variable of the old function type must be updated

### Fixed
- GoDoc claimed the command prefix is stripped from `ExampleInfo.Usage`; it is kept verbatim, as it always was
- GoDoc claimed `MinIndent: 0` disables indentation detection; it selects the default of 2 (use `DisableIndentDetection`)
- GoDoc and README claimed `.Files` is sorted alphabetically; it follows Cobra's command order
- README listed a non-existent `headingLen` helper and misdescribed `replaceSpaces`

## [0.2.1] - 2026-03-16

### Changed
- CI hardening only: SAST workflows (gosec, govulncheck), zizmor workflow lint, pinned action versions, `persist-credentials: false`; maintainer release process documented in README. No library changes.

## [0.2.0] - 2026-02-05

### Added
- Panic recovery around template execution to turn panics into errors
- Concurrent usage test coverage and `make test-race` target
- Comprehensive GoDoc for exported types (`FlagInfo`, `ExampleInfo`, `ExampleParser`, `TemplateInfo`)

### Changed
- **BREAKING**: Removed the unused `linkHandler` parameter from `GenDocs`, `GenMarkdownTree`, and `GenRSTTree`; link formatting now lives entirely in templates

## [0.1.3] - 2026-01-14

### Added
- Input validation for `TemplateInfo` fields with descriptive error messages
- Auto-creation of output directories (including nested paths)

## [0.1.2] - 2026-01-14

### Added
- `GenRSTTree()` and `GenMarkdownTree()` for format-specific documentation generation
- Custom GitHub Copilot agents for development guidance

### Changed
- **BREAKING**: `GenDocsTree()` replaced by `GenRSTTree()` and `GenMarkdownTree()`
- **BREAKING**: Command ordering now preserves Cobra's order instead of alphabetical sorting

### Fixed
- Loop variable capture in parallel tests

## [0.1.1] - 2025-02-28

### Added
- Comprehensive test suite for reST and Markdown template examples
- Makefile for build, test, and dependency management
- LGPL-3.0-only license with proper SPDX headers
- README documentation improvements

### Changed
- Improved example parsing with configurable `ExampleParser`
- Enhanced template testing patterns

## [0.1.0] - 2025-02-28

### Changed
- Relaxed Go version requirement to support broader compatibility

## [0.0.8] - 2025-02-27

### Changed
- Renamed internal functions for clarity and consistency

## [0.0.7] - 2025-02-27

### Changed
- Implemented code review feedback
- Internal refactoring and improvements

## [0.0.6] - 2025-02-27

### Added
- Development tooling and test infrastructure
- Initial test coverage

## [0.0.5] - 2025-02-27

_No documented changes available for this version._

## [0.0.4] - 2025-02-27

### Added
- Custom related links support via command `Annotations`
- Test suite improvements

### Changed
- Cleaned up codebase
- Renamed example templates for clarity

## [0.0.3] - 2025-02-27

### Changed
- Moved default templates to `examples/` directory

## [0.0.2] - 2025-02-27

### Added
- Initial template functionality
- Basic Cobra command documentation extraction

## [0.0.1] - 2025-02-27

### Added
- Initial release
- Basic documentation generation from Cobra CLI applications
- Support for custom Go templates
- Example extraction and flag documentation

[Unreleased]: https://github.com/canonical/gencodo/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/canonical/gencodo/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/canonical/gencodo/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/canonical/gencodo/compare/v0.1.3...v0.2.0
[0.1.3]: https://github.com/canonical/gencodo/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/canonical/gencodo/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/canonical/gencodo/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/canonical/gencodo/compare/v0.0.8...v0.1.0
[0.0.8]: https://github.com/canonical/gencodo/compare/v0.0.7...v0.0.8
[0.0.7]: https://github.com/canonical/gencodo/compare/v0.0.6...v0.0.7
[0.0.6]: https://github.com/canonical/gencodo/compare/v0.0.5...v0.0.6
[0.0.5]: https://github.com/canonical/gencodo/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/canonical/gencodo/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/canonical/gencodo/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/canonical/gencodo/releases/tag/v0.0.2
