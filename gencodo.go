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
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FlagInfo holds metadata about a CLI flag.
type FlagInfo struct {
	Name         string // Flag name
	Usage        string // Description of the flag
	DefaultValue string // Default value of the flag
}

// ExampleInfo represents an example usage of a command to be used in templates.
type ExampleInfo struct {
	Info  string // Description of the example
	Usage string // Example command usage
}

// ExampleParser configures how examples are parsed from command example strings.
type ExampleParser struct {
	CommandPrefixes []string // Prefixes that indicate command lines (e.g., "$", ">", "#")
	MinIndent       int      // Minimum indentation to consider a line as a command
}

// TemplateInfo stores templates used for documentation generation.
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

// GenDocs generates docs for a single command in an eponymous file.
func GenDocs(cmd *cobra.Command, w io.Writer, templateContent string, linkHandler func(string, string) string) error {
	if cmd == nil {
		return fmt.Errorf("command cannot be nil")
	}
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}

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

	tmpl, err := template.New("command").Funcs(funcMap).Parse(templateContent)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err = tmpl.Execute(buf, data); err != nil {
		return err
	}

	_, err = buf.WriteTo(w)
	return err
}

// GenDocsTree generates docs for a subcommand tree, skipping the root.
func GenDocsTree(
	cmd *cobra.Command,
	dir string,
	templates TemplateInfo,
	filePrepender func(string) string,
	linkHandler func(string, string) string,
) error {
	if cmd == nil {
		return fmt.Errorf("command cannot be nil")
	}

	// Validate template configuration
	if err := templates.Validate(); err != nil {
		return fmt.Errorf("invalid template configuration: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var files []string

	var generateDocs func(*cobra.Command) error
	generateDocs = func(c *cobra.Command) error {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			return nil
		}

		for _, subCmd := range c.Commands() {
			if err := generateDocs(subCmd); err != nil {
				return err
			}
		}

		basename := strings.ReplaceAll(c.CommandPath(), " ", "-") + ".rst"
		filename := filepath.Join(dir, basename)
		f, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.WriteString(f, filePrepender(filename)); err != nil {
			return err
		}
		if err := GenDocs(c, f, templates.SingleCommandTemplate, linkHandler); err != nil {
			return err
		}

		files = append(files, basename)
		return nil
	}

	for _, subCmd := range cmd.Commands() {
		if err := generateDocs(subCmd); err != nil {
			return err
		}
	}

	sort.Strings(files)

	data := struct {
		Files []string
	}{
		Files: files,
	}

	tmpl, err := template.New("index").Parse(templates.IndexTemplate)
	if err != nil {
		return err
	}

	indexPath := filepath.Join(dir, templates.IndexFileName)
	indexFile, err := os.Create(indexPath)
	if err != nil {
		return err
	}
	defer indexFile.Close()

	if err := tmpl.Execute(indexFile, data); err != nil {
		return err
	}

	return nil
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
