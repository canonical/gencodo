# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
