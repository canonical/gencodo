// This file is part of gencodo, a library for generating Go template based docs from cobra CLI applications
//
// Copyright 2025 Canonical Ltd.
//
// SPDX-License-Identifier: LGPL-3.0-only
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License version 3,
// as published by the Free Software Foundation.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranties of
// MERCHANTABILITY, SATISFACTORY QUALITY, or FITNESS FOR A PARTICULAR PURPOSE.
// See the GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see http://www.gnu.org/licenses/.

package gencodo

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FlagInfo holds metadata about a CLI flag extracted from Cobra commands.
// Only non-inherited flags are included (flags defined directly on the command,
// not persistent flags from parent commands).
//
// Example usage in templates:
//
//	{{- range .Flags }}
//	--{{ .Name }}    {{ .Usage }}
//	  Default: {{ .DefaultValue }}
//	{{- end }}
//
// The DefaultValue field contains the flag's string representation as shown
// in help text (e.g., "false" for booleans, "[]" for empty slices).
type FlagInfo struct {
	Name         string // Flag name
	Usage        string // Description of the flag
	DefaultValue string // Default value of the flag
}

// ExampleInfo represents a parsed example from a Cobra command's example string.
// Examples are split into separate Info (description) and Usage (command) fields
// based on the ExampleParser configuration.
//
// Cobra's Example field typically contains multi-line strings like:
//
//	"Display help for this command:\n  $ myapp help\n\nList available resources:\n  $ myapp list"
//
// ExampleParser splits this by double newlines (\n\n) and identifies command lines
// via prefixes (e.g., "$") or indentation. The result is:
//
//	[]ExampleInfo{
//	    {Info: "Display help for this command:", Usage: "myapp help"},
//	    {Info: "List available resources:", Usage: "myapp list"},
//	}
//
// If no command line is detected, the entire block becomes Usage with empty Info.
type ExampleInfo struct {
	Info  string // Description of the example
	Usage string // Example command usage
}

// ExampleParser configures how examples are parsed from Cobra command example strings.
// It splits example text by double newlines (\n\n) and identifies command lines
// using prefixes or indentation heuristics.
//
// Default behavior (zero values):
//   - CommandPrefixes: ["$", ">", "#"] (common shell prompts)
//   - MinIndent: 2 (lines indented ≥2 spaces are considered commands)
//
// The parser first checks for CommandPrefixes. If a line starts with any prefix
// (after trimming leading whitespace), it's treated as a command. The prefix and
// following space are removed from the Usage field.
//
// If no prefix matches and MinIndent > 0, lines with indentation ≥ MinIndent are
// considered commands. Indentation is preserved in the Usage field.
//
// Example - Shell commands only:
//
//	parser := ExampleParser{CommandPrefixes: []string{"$"}, MinIndent: 0}
//	examples := parser.Parse(cmd.Example)
//
// Example - Indented code blocks without prefixes:
//
//	parser := ExampleParser{MinIndent: 4}
//	examples := parser.Parse(cmd.Example)
//
// Example - Windows PowerShell prompts:
//
//	parser := ExampleParser{CommandPrefixes: []string{"PS>", "C:\\>"}}
//	examples := parser.Parse(cmd.Example)
type ExampleParser struct {
	CommandPrefixes []string // Prefixes that indicate command lines (e.g., "$", ">", "#")
	MinIndent       int      // Minimum indentation to consider a line as a command
}

// TemplateInfo stores Go templates for generating documentation from Cobra commands.
// All fields are required and validated via the Validate() method.
//
// Templates use Go's text/template syntax and have access to different data structures:
//
// IndexTemplate receives:
//   - .Files ([]string): Alphabetically sorted list of generated command filenames
//
// SingleCommandTemplate receives:
//   - .Ref (string): Reference ID (command name with spaces → underscores)
//   - .CommandName (string): Full command path (e.g., "myapp subcommand")
//   - .Short (string): Short description
//   - .Long (string): Long description (falls back to .Short if empty)
//   - .Synopsis (string): Command usage line from Cobra
//   - .Examples ([]ExampleInfo): Parsed examples
//   - .Flags ([]FlagInfo): Non-inherited flags only
//   - .HeadingLen (int): Length of CommandName (for formatting)
//   - .RelatedCommands ([]string): From Annotations["related"] (comma-separated)
//
// Example - Simple Markdown template:
//
//	templateInfo := TemplateInfo{
//	    IndexFileName: "index.md",
//	    IndexTemplate: "# Commands\n{{range .Files}}- [{{.}}]({{.}})\n{{end}}",
//	    SingleCommandTemplate: "# {{.CommandName}}\n\n{{.Long}}\n",
//	}
//	if err := templateInfo.Validate(); err != nil {
//	    return err
//	}
//
// For complete examples, see examples/command.md and examples/command.rst in the repository.
type TemplateInfo struct {
	IndexFileName         string // Name of the generated index file
	IndexTemplate         string // Template for the index file
	SingleCommandTemplate string // Template for individual command files
}

// Validate checks that all required TemplateInfo fields are non-empty.
func (t *TemplateInfo) Validate() error {
	if t.IndexFileName == "" {
		return fmt.Errorf("TemplateInfo.IndexFileName cannot be empty")
	}
	if t.IndexTemplate == "" {
		return fmt.Errorf("TemplateInfo.IndexTemplate cannot be empty")
	}
	if t.SingleCommandTemplate == "" {
		return fmt.Errorf("TemplateInfo.SingleCommandTemplate cannot be empty")
	}
	return nil
}

// format is the output format for generated docs.
type format int

const (
	formatMarkdown format = iota
	formatRST
)

func (f format) FileExtension() string {
	switch f {
	case formatMarkdown:
		return ".md"
	case formatRST:
		return ".rst"
	default:
		panic(fmt.Sprintf("unknown format: %d", f))
	}
}

// GenDocs generates docs for a single command in an eponymous file.
func GenDocs(cmd *cobra.Command, w io.Writer, templateContent string) error {
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	name := cmd.CommandPath()

	short := cmd.Short
	long := cmd.Long
	if len(long) == 0 {
		long = short
	}
	ref := "ref_" + strings.ReplaceAll(name, " ", "_")

	headinglen := len(name)

	// Use default parser with common command prefixes
	parser := ExampleParser{
		CommandPrefixes: []string{"$", ">", "#"},
		MinIndent:       2,
	}
	structuredExamples := parser.Parse(cmd.Example)

	flags := cmd.NonInheritedFlags()
	var FlagDetails []FlagInfo
	flags.VisitAll(func(flag *pflag.Flag) {
		FlagDetails = append(FlagDetails, FlagInfo{
			Name:         flag.Name,
			Usage:        flag.Usage,
			DefaultValue: flag.DefValue,
		})
	})

	// Extract related commands from Annotations
	relatedCommands := []string{}
	if related, exists := cmd.Annotations["related"]; exists {
		relatedCommands = strings.Split(related, ",")
		for i, cmd := range relatedCommands {
			relatedCommands[i] = strings.TrimSpace(cmd)
		}
	}

	data := struct {
		Ref             string
		CommandName     string
		Short           string
		Long            string
		Synopsis        string
		Examples        []ExampleInfo
		Flags           []FlagInfo
		HeadingLen      int // CommandName length
		RelatedCommands []string
	}{
		Ref:             ref,
		CommandName:     name,
		Short:           short,
		Long:            long,
		Synopsis:        cmd.UseLine(),
		Examples:        structuredExamples,
		Flags:           FlagDetails,
		HeadingLen:      headinglen,
		RelatedCommands: relatedCommands,
	}

	// Basic utility functions.
	funcMap := template.FuncMap{
		"indent": func(spaces int, ss ...string) string {
			padding := strings.Repeat(" ", spaces)
			var indentedStrings []string
			for _, s := range ss {
				indentedStrings = append(indentedStrings, padding+strings.ReplaceAll(s, "\n", "\n"+padding))
			}
			return strings.Join(indentedStrings, "\n")
		},
		"repeat": strings.Repeat,
		"replaceSpaces": func(s string) string {
			return strings.ReplaceAll(s, " ", "_")
		},
	}

	tmpl, err := template.New("command").
		Option("missingkey=error").
		Funcs(funcMap).
		Parse(templateContent)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err = executeTemplate(tmpl, buf, data); err != nil {
		return err
	}

	_, err = buf.WriteTo(w)
	return err
}

func GenMarkdownTree(
	cmd *cobra.Command,
	dir string,
	templates TemplateInfo,
	filePrepender func(string) string,
) error {
	return genDocsTree(cmd, dir, templates, filePrepender, formatMarkdown)
}

func GenRSTTree(
	cmd *cobra.Command,
	dir string,
	templates TemplateInfo,
	filePrepender func(string) string,
) error {
	return genDocsTree(cmd, dir, templates, filePrepender, formatRST)
}

// genDocsTree generates docs for a subcommand tree, skipping the root.
func genDocsTree(
	cmd *cobra.Command,
	dir string,
	templates TemplateInfo,
	filePrepender func(string) string,
	format format,
) error {
	// Validate template configuration
	if err := templates.Validate(); err != nil {
		return fmt.Errorf("invalid template configuration: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var files []string

	var generateDocs func(*cobra.Command) error
	generateDocs = func(c *cobra.Command) error {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			return nil
		}

		basename := strings.ReplaceAll(c.CommandPath(), " ", "-") + format.FileExtension()
		filename := filepath.Join(dir, basename)
		f, err := os.Create(filepath.Clean(filename))
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.WriteString(f, filePrepender(filename)); err != nil {
			return err
		}
		if err := GenDocs(c, f, templates.SingleCommandTemplate); err != nil {
			return err
		}

		files = append(files, basename)

		for _, subCmd := range c.Commands() {
			if err := generateDocs(subCmd); err != nil {
				return err
			}
		}

		return nil
	}

	for _, subCmd := range cmd.Commands() {
		if err := generateDocs(subCmd); err != nil {
			return err
		}
	}

	data := struct {
		Files []string
	}{
		Files: files,
	}

	tmpl, err := template.New("index").
		Option("missingkey=error").
		Parse(templates.IndexTemplate)
	if err != nil {
		return err
	}

	indexPath := filepath.Join(dir, templates.IndexFileName)
	indexFile, err := os.Create(filepath.Clean(indexPath))
	if err != nil {
		return err
	}
	defer indexFile.Close()

	if err := executeTemplate(tmpl, indexFile, data); err != nil {
		return err
	}

	return nil
}

// executeTemplate safely executes a template with panic recovery.
// It recovers from panics during template execution and converts them to errors.
func executeTemplate(tmpl *template.Template, w io.Writer, data interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("template execution panic: %v", r)
		}
	}()
	err = tmpl.Execute(w, data)
	return err
}

// Parse extracts structured examples from a raw example string.
// It uses a state machine approach with configurable command detection.
func (p *ExampleParser) Parse(example string) []ExampleInfo {
	if example == "" {
		return nil
	}

	// Set defaults if not configured
	prefixes := p.CommandPrefixes
	if len(prefixes) == 0 {
		prefixes = []string{"$", ">", "#"}
	}
	minIndent := p.MinIndent
	if minIndent <= 0 {
		minIndent = 2
	}

	// Split by double newlines to get separate example blocks
	entries := strings.Split(example, "\n\n")
	var results []ExampleInfo

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		lines := strings.Split(entry, "\n")
		var infoLines, usageLines []string
		foundCommand := false

		for i, line := range lines {
			// Check if this line looks like a command
			isCommand := p.isCommandLine(line, prefixes, minIndent)

			if isCommand && !foundCommand {
				// First command line found - split here
				infoLines = lines[:i]
				usageLines = lines[i:]
				foundCommand = true
				break
			}
		}

		// Fallback: if no command prefix/indent detected, treat as usage-only
		if !foundCommand && len(lines) > 0 {
			usageLines = lines
		}

		// Add example if we have usage content
		if len(usageLines) > 0 {
			info := ""
			if len(infoLines) > 0 {
				// Join and trim info lines
				info = strings.TrimSpace(strings.Join(infoLines, "\n"))
			}

			results = append(results, ExampleInfo{
				Info:  info,
				Usage: strings.Join(usageLines, "\n"),
			})
		}
	}

	return results
}

// isCommandLine determines if a line looks like a command based on prefixes or indentation.
func (p *ExampleParser) isCommandLine(line string, prefixes []string, minIndent int) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	// Check for command prefixes
	for _, prefix := range prefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}

	// Check for indentation (commands are often indented in examples)
	leadingSpaces := len(line) - len(strings.TrimLeft(line, " \t"))
	if leadingSpaces >= minIndent && trimmed != "" {
		return true
	}

	return false
}
