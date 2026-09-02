// This file is part of gencodo, a library for generating Go template based docs from cobra CLI applications
//
// Copyright 2025-2026 Canonical Ltd.
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
	"sync"
	"testing"
	"text/template"

	"github.com/spf13/cobra"
)

func fileExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	if err != nil {
		t.Fatalf("%s not created: %v", filepath.Base(path), err)
	}
}

func readDirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read %s: %v", dir, err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func containsAll(all []string, needed []string) bool {
	m := map[string]bool{}
	for _, a := range all {
		m[a] = true
	}
	for _, n := range needed {
		if !m[n] {
			return false
		}
	}
	return true
}

func TestGenDocs(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "example",
		Short: "An example command",
		Long:  "A longer description of the example command.",
		Example: `
Foo:
$ example foo

Bar:
$ example bar`,
		Annotations: map[string]string{
			"related": "foo bar,bar,baz",
		},
	}
	var output bytes.Buffer
	templateContent := `{{ .CommandName }}|{{ .Short }}|{{ .Long }}{{ range .Examples }}|{{ .Info }}|{{ .Usage }}{{ end }}{{ range .RelatedCommands }}|{{ . | replaceSpaces }}{{ end }}`
	err := GenDocs(cmd, &output, templateContent)
	if err != nil {
		t.Fatalf("GenDocs failed: %v", err)
	}
	expected := "example|An example command|A longer description of the example command.|Foo:|$ example foo|Bar:|$ example bar|foo_bar|bar|baz"
	if output.String() != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, output.String())
	}
}

func TestGenDocsEmptyExample(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "noexample",
		Short: "No example command",
	}
	var output bytes.Buffer
	templateContent := `{{ .CommandName }}|{{ .Short }}|{{ .Long }}|`
	err := GenDocs(cmd, &output, templateContent)
	if err != nil {
		t.Fatalf("GenDocs failed: %v", err)
	}
	if !strings.Contains(output.String(), "noexample|No example command|No example command|") {
		t.Errorf("unexpected output: %s", output.String())
	}
}

func TestGenDocsIndentRepeat(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "indentrepeat",
		Short: "Testing indent and repeat",
	}
	var output bytes.Buffer
	templateContent := `{{ indent 2 .CommandName }}{{ repeat "X" 3 }}`
	err := GenDocs(cmd, &output, templateContent)
	if err != nil {
		t.Fatalf("GenDocs failed: %v", err)
	}
	if !strings.Contains(output.String(), "  indentrepeatXXX") {
		t.Errorf("unexpected template output: %s", output.String())
	}
}

func TestGenDocTree(t *testing.T) {
	tests := map[string]struct {
		genDocFunc    func(*cobra.Command, string, TemplateInfo, func(string) string, ...Option) error
		fileExtension string
	}{
		"TestGenRSTTree":      {GenRSTTree, ".rst"},
		"TestGenMarkdownTree": {GenMarkdownTree, ".md"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			rootCmd := &cobra.Command{Use: "root"}
			subCmd1 := &cobra.Command{
				Use:   "sub1",
				Short: "Subcommand 1",
				Run:   func(cmd *cobra.Command, args []string) {},
			}
			subCmd2 := &cobra.Command{
				Use:   "sub2",
				Short: "Subcommand 2",
				Run:   func(cmd *cobra.Command, args []string) {},
			}
			rootCmd.AddCommand(subCmd1, subCmd2)
			templates := TemplateInfo{
				IndexFileName:         "root" + tc.fileExtension,
				IndexTemplate:         `{{ range .Files }}{{ . }}\n{{ end }}`,
				SingleCommandTemplate: `{{ .CommandName }} - {{ .Short }}`,
			}
			err := tc.genDocFunc(rootCmd, tempDir, templates, func(s string) string { return "" })
			if err != nil {
				t.Fatalf("genDocFunc failed: %v", err)
			}
			fileExists(t, filepath.Join(tempDir, "root"+tc.fileExtension))
			fileExists(t, filepath.Join(tempDir, "root-sub1"+tc.fileExtension))
			fileExists(t, filepath.Join(tempDir, "root-sub2"+tc.fileExtension))
		})
	}
}

func TestGenDocTreeNested(t *testing.T) {
	tests := map[string]struct {
		genDocFunc    func(*cobra.Command, string, TemplateInfo, func(string) string, ...Option) error
		fileExtension string
	}{
		"TestGenRSTTree":      {GenRSTTree, ".rst"},
		"TestGenMarkdownTree": {GenMarkdownTree, ".md"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			rootCmd := &cobra.Command{Use: "top"}
			subCmdA := &cobra.Command{Use: "A", Short: "A", Run: func(cmd *cobra.Command, args []string) {}}
			subCmdB := &cobra.Command{Use: "B", Short: "B", Run: func(cmd *cobra.Command, args []string) {}}
			subCmdC := &cobra.Command{Use: "C", Short: "C", Run: func(cmd *cobra.Command, args []string) {}}
			subCmdA1 := &cobra.Command{Use: "A1", Short: "A1", Run: func(cmd *cobra.Command, args []string) {}}
			subCmdA.AddCommand(subCmdA1)
			rootCmd.AddCommand(subCmdA, subCmdB, subCmdC)
			templates := TemplateInfo{
				IndexFileName:         "top" + tc.fileExtension,
				IndexTemplate:         `{{ range .Files }}{{ . }}\n{{ end }}`,
				SingleCommandTemplate: `{{ .CommandName }} - {{ .Short }}`,
			}
			err := tc.genDocFunc(rootCmd, tempDir, templates, func(s string) string { return "" })
			if err != nil {
				t.Fatalf("GenRSTTree failed: %v", err)
			}
			names := readDirNames(t, tempDir)
			var expectedFiles []string
			for _, basename := range []string{"top", "top-A", "top-A-A1", "top-B", "top-C"} {
				expectedFiles = append(expectedFiles, basename+tc.fileExtension)
			}
			if !containsAll(names, expectedFiles) {
				t.Errorf("unexpected files: %v", names)
			}
		})
	}
}

func TestGenDocsTreeNonExistentDir(t *testing.T) {
	tempDir := t.TempDir()
	// Create a nested path that doesn't exist
	outputDir := filepath.Join(tempDir, "docs", "cli", "commands")

	rootCmd := &cobra.Command{Use: "app"}
	subCmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
	rootCmd.AddCommand(subCmd)

	templates := TemplateInfo{
		IndexFileName:         "index.rst",
		IndexTemplate:         `{{ range .Files }}{{ . }}{{ end }}`,
		SingleCommandTemplate: `{{ .CommandName }}`,
	}

	// Should succeed even though directory doesn't exist
	err := GenRSTTree(rootCmd, outputDir, templates, func(s string) string { return "" })
	if err != nil {
		t.Fatalf("GenRSTTree should create directory, but failed: %v", err)
	}

	// Verify directory was created and files exist
	fileExists(t, filepath.Join(outputDir, "index.rst"))
	fileExists(t, filepath.Join(outputDir, "app-test.rst"))
}

func TestGenDocsTreeEmptyTemplateInfo(t *testing.T) {
	tempDir := t.TempDir()
	rootCmd := &cobra.Command{Use: "test"}
	subCmd := &cobra.Command{Use: "sub", Short: "Sub", Run: func(cmd *cobra.Command, args []string) {}}
	rootCmd.AddCommand(subCmd)

	tests := []struct {
		name      string
		templates TemplateInfo
		wantErr   string
	}{
		{
			name: "empty IndexFileName",
			templates: TemplateInfo{
				IndexFileName:         "",
				IndexTemplate:         `{{ range .Files }}{{ . }}{{ end }}`,
				SingleCommandTemplate: `{{ .CommandName }}`,
			},
			wantErr: "TemplateInfo.IndexFileName cannot be empty",
		},
		{
			name: "empty IndexTemplate",
			templates: TemplateInfo{
				IndexFileName:         "index.rst",
				IndexTemplate:         "",
				SingleCommandTemplate: `{{ .CommandName }}`,
			},
			wantErr: "TemplateInfo.IndexTemplate cannot be empty",
		},
		{
			name: "empty SingleCommandTemplate",
			templates: TemplateInfo{
				IndexFileName:         "index.rst",
				IndexTemplate:         `{{ range .Files }}{{ . }}{{ end }}`,
				SingleCommandTemplate: "",
			},
			wantErr: "TemplateInfo.SingleCommandTemplate cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenRSTTree(rootCmd, tempDir, tt.templates, func(s string) string { return "" })
			if err == nil {
				t.Fatal("expected error for empty template field, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing '%s', got '%s'", tt.wantErr, err.Error())
			}
		})
	}
}

func TestGenDocsFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "flagtest",
		Short: "Command with flags",
	}
	cmd.Flags().String("output", "stdout", "Output destination")
	cmd.Flags().Int("count", 5, "Number of iterations")
	cmd.Flags().Bool("verbose", false, "Enable verbose mode")

	var output bytes.Buffer
	templateContent := `{{ range .Flags }}{{ .Name }}:{{ .Usage }}:{{ .DefaultValue }}|{{ end }}`
	err := GenDocs(cmd, &output, templateContent)
	if err != nil {
		t.Fatalf("GenDocs failed: %v", err)
	}

	result := output.String()
	if !strings.Contains(result, "output:Output destination:stdout|") {
		t.Errorf("output flag not found in: %s", result)
	}
	if !strings.Contains(result, "count:Number of iterations:5|") {
		t.Errorf("count flag not found in: %s", result)
	}
	if !strings.Contains(result, "verbose:Enable verbose mode:false|") {
		t.Errorf("verbose flag not found in: %s", result)
	}
}

func TestGenDocsSynopsis(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "syncmd [flags]",
		Short: "Test synopsis",
	}
	var output bytes.Buffer
	templateContent := `{{ .Synopsis }}`
	err := GenDocs(cmd, &output, templateContent)
	if err != nil {
		t.Fatalf("GenDocs failed: %v", err)
	}

	if !strings.Contains(output.String(), "syncmd [flags]") {
		t.Errorf("Synopsis not rendered correctly: %s", output.String())
	}
}

func TestGenDocsRef(t *testing.T) {
	rootCmd := &cobra.Command{Use: "root"}
	subCmd := &cobra.Command{
		Use:   "multi word",
		Short: "Command with spaces",
	}
	rootCmd.AddCommand(subCmd)

	var output bytes.Buffer
	templateContent := `{{ .Ref }}`
	err := GenDocs(subCmd, &output, templateContent)
	if err != nil {
		t.Fatalf("GenDocs failed: %v", err)
	}

	expected := "ref_root_multi"
	if output.String() != expected {
		t.Errorf("expected Ref %s, got %s", expected, output.String())
	}
}

func TestGenDocsLongFallback(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "shortonly",
		Short: "Short description only",
	}
	var output bytes.Buffer
	templateContent := `{{ .Short }}|{{ .Long }}`
	err := GenDocs(cmd, &output, templateContent)
	if err != nil {
		t.Fatalf("GenDocs failed: %v", err)
	}

	expected := "Short description only|Short description only"
	if output.String() != expected {
		t.Errorf("expected %s, got %s", expected, output.String())
	}
}

func TestGenDocsFilePrepender(t *testing.T) {
	tempDir := t.TempDir()
	rootCmd := &cobra.Command{Use: "root"}
	subCmd := &cobra.Command{
		Use:   "sub",
		Short: "Subcommand",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
	rootCmd.AddCommand(subCmd)

	headerContent := "# Auto-generated file\n"
	templates := TemplateInfo{
		IndexFileName:         "root.rst",
		IndexTemplate:         `{{ range .Files }}{{ . }}{{ end }}`,
		SingleCommandTemplate: `Content`,
	}

	err := GenRSTTree(rootCmd, tempDir, templates, func(s string) string { return headerContent })
	if err != nil {
		t.Fatalf("GenRSTTree failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tempDir, "root-sub.rst"))
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	if !strings.HasPrefix(string(content), headerContent) {
		t.Errorf("file does not start with prepended content: %s", string(content))
	}
}

func TestGenDocsIndexContent(t *testing.T) {
	tempDir := t.TempDir()
	rootCmd := &cobra.Command{Use: "app"}
	cmd1 := &cobra.Command{Use: "alpha", Short: "Alpha", Run: func(cmd *cobra.Command, args []string) {}}
	cmd2 := &cobra.Command{Use: "beta", Short: "Beta", Run: func(cmd *cobra.Command, args []string) {}}
	rootCmd.AddCommand(cmd1, cmd2)

	templates := TemplateInfo{
		IndexFileName:         "index.rst",
		IndexTemplate:         `Files:\n{{ range .Files }}- {{ . }}\n{{ end }}`,
		SingleCommandTemplate: `{{ .CommandName }}`,
	}

	err := GenRSTTree(rootCmd, tempDir, templates, func(s string) string { return "" })
	if err != nil {
		t.Fatalf("GenRSTTree failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tempDir, "index.rst"))
	if err != nil {
		t.Fatalf("failed to read index file: %v", err)
	}

	indexContent := string(content)
	if !strings.Contains(indexContent, "- app-alpha.rst") {
		t.Errorf("index missing app-alpha.rst: %s", indexContent)
	}
	if !strings.Contains(indexContent, "- app-beta.rst") {
		t.Errorf("index missing app-beta.rst: %s", indexContent)
	}
}

func TestGenDocsInvalidTemplate(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}
	var output bytes.Buffer
	invalidTemplate := `{{ .CommandName `
	err := GenDocs(cmd, &output, invalidTemplate)
	if err == nil {
		t.Fatal("expected error for invalid template, got nil")
	}
}

func TestGenRSTTreeInvalidIndexTemplate(t *testing.T) {
	tempDir := t.TempDir()
	rootCmd := &cobra.Command{Use: "root"}
	subCmd := &cobra.Command{Use: "sub", Short: "Sub", Run: func(cmd *cobra.Command, args []string) {}}
	rootCmd.AddCommand(subCmd)

	templates := TemplateInfo{
		IndexFileName:         "index.rst",
		IndexTemplate:         `{{ .Files `,
		SingleCommandTemplate: `{{ .CommandName }}`,
	}

	err := GenRSTTree(rootCmd, tempDir, templates, func(s string) string { return "" })
	if err == nil {
		t.Fatal("expected error for invalid index template, got nil")
	}
}

func TestGenDocsPanicRecovery(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	tests := []struct {
		name     string
		template string
	}{
		{
			name:     "missing field access",
			template: `{{ .NonExistentField }}`,
		},
		{
			name:     "nested missing field",
			template: `{{ .Foo.Bar.Baz }}`,
		},
		{
			name:     "method on missing field",
			template: `{{ .NonExistent.Method }}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			err := GenDocs(cmd, &output, tt.template)
			if err == nil {
				t.Error("expected error for template accessing non-existent field, got nil")
			}
			if !strings.Contains(err.Error(), "can't evaluate field") && !strings.Contains(err.Error(), "panic") {
				t.Errorf("expected error message about field evaluation, got: %v", err)
			}
		})
	}
}

func TestGenDocsTreePanicRecovery(t *testing.T) {
	tempDir := t.TempDir()
	rootCmd := &cobra.Command{Use: "root"}
	subCmd := &cobra.Command{Use: "sub", Short: "Sub", Run: func(cmd *cobra.Command, args []string) {}}
	rootCmd.AddCommand(subCmd)

	tests := []struct {
		name          string
		indexTemplate string
		cmdTemplate   string
	}{
		{
			name:          "invalid index template field",
			indexTemplate: `{{ .NonExistentField }}`,
			cmdTemplate:   `{{ .CommandName }}`,
		},
		{
			name:          "invalid command template field",
			indexTemplate: `{{ range .Files }}{{ . }}{{ end }}`,
			cmdTemplate:   `{{ .Foo.Bar }}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templates := TemplateInfo{
				IndexFileName:         "index.rst",
				IndexTemplate:         tt.indexTemplate,
				SingleCommandTemplate: tt.cmdTemplate,
			}

			err := GenRSTTree(rootCmd, tempDir, templates, func(s string) string { return "" })
			if err == nil {
				t.Error("expected error for template with non-existent field, got nil")
			}
		})
	}
}

func TestGenDocsReplaceSpacesFunction(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test",
		Annotations: map[string]string{
			"related": "foo bar,baz qux",
		},
	}
	var output bytes.Buffer
	templateContent := `{{ range .RelatedCommands }}{{ . | replaceSpaces }}|{{ end }}`
	err := GenDocs(cmd, &output, templateContent)
	if err != nil {
		t.Fatalf("GenDocs failed: %v", err)
	}

	expected := "foo_bar|baz_qux|"
	if output.String() != expected {
		t.Errorf("expected %s, got %s", expected, output.String())
	}
}

func TestGenDocsHeadingLen(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "longcommandname",
		Short: "Test",
	}
	var output bytes.Buffer
	templateContent := `{{ .HeadingLen }}`
	err := GenDocs(cmd, &output, templateContent)
	if err != nil {
		t.Fatalf("GenDocs failed: %v", err)
	}

	if output.String() != "15" {
		t.Errorf("expected HeadingLen 15, got %s", output.String())
	}
}

func TestGenDocsNoRelatedCommands(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test without related commands",
	}
	var output bytes.Buffer
	templateContent := `{{ len .RelatedCommands }}`
	err := GenDocs(cmd, &output, templateContent)
	if err != nil {
		t.Fatalf("GenDocs failed: %v", err)
	}

	if output.String() != "0" {
		t.Errorf("expected 0 related commands, got %s", output.String())
	}
}

func TestGenDocsNonInheritedFlags(t *testing.T) {
	rootCmd := &cobra.Command{
		Use:   "root",
		Short: "Root command",
	}
	rootCmd.PersistentFlags().String("config", "", "Config file")

	subCmd := &cobra.Command{
		Use:   "sub",
		Short: "Sub command",
	}
	subCmd.Flags().String("local", "", "Local flag")
	rootCmd.AddCommand(subCmd)

	var output bytes.Buffer
	templateContent := `{{ range .Flags }}{{ .Name }}|{{ end }}`
	err := GenDocs(subCmd, &output, templateContent)
	if err != nil {
		t.Fatalf("GenDocs failed: %v", err)
	}

	result := output.String()
	if !strings.Contains(result, "local|") {
		t.Errorf("local flag should be included: %s", result)
	}
	if strings.Contains(result, "config") {
		t.Errorf("inherited flag should not be included: %s", result)
	}
}

func TestReStructuredTextTemplates(t *testing.T) {
	tempDir := t.TempDir()

	rootCmd := &cobra.Command{
		Use:   "myproject",
		Short: "A sample CLI tool",
		Long:  "myproject is a sample CLI tool for testing reStructuredText templates",
	}

	subCmd := &cobra.Command{
		Use:   "process [file]",
		Short: "Process a file",
		Long:  "Process a file and perform various transformations on it",
		Example: `Basic usage:
$ myproject process input.txt

With options:
$ myproject process --format json input.txt`,
		Run: func(cmd *cobra.Command, args []string) {},
	}
	subCmd.Flags().String("format", "text", "Output format")
	subCmd.Flags().Bool("verbose", false, "Enable verbose output")

	rootCmd.AddCommand(subCmd)

	indexTemplate, err := os.ReadFile("examples/cli.rst")
	if err != nil {
		t.Fatalf("failed to read index template: %v", err)
	}
	commandTemplate, err := os.ReadFile("examples/command.rst")
	if err != nil {
		t.Fatalf("failed to read command template: %v", err)
	}

	templates := TemplateInfo{
		IndexFileName:         "myproject-cli.rst",
		IndexTemplate:         string(indexTemplate),
		SingleCommandTemplate: string(commandTemplate),
	}

	err = GenRSTTree(rootCmd, tempDir, templates,
		func(s string) string { return "" })
	if err != nil {
		t.Fatalf("GenRSTTree failed: %v", err)
	}

	indexPath := filepath.Join(tempDir, "myproject-cli.rst")
	fileExists(t, indexPath)
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index file: %v", err)
	}

	indexStr := string(indexContent)
	if !strings.Contains(indexStr, ".. _ref_myproject_cli:") {
		t.Errorf("index missing reference label: %s", indexStr)
	}
	if !strings.Contains(indexStr, "myproject (CLI)") {
		t.Errorf("index missing header: %s", indexStr)
	}
	if !strings.Contains(indexStr, ".. include:: myproject-process.rst") {
		t.Errorf("index missing process command include: %s", indexStr)
	}

	processPath := filepath.Join(tempDir, "myproject-process.rst")
	fileExists(t, processPath)
	processContent, err := os.ReadFile(processPath)
	if err != nil {
		t.Fatalf("failed to read process file: %v", err)
	}

	processStr := string(processContent)
	if !strings.Contains(processStr, ".. _ref_myproject_process:") {
		t.Errorf("process file missing reference label: %s", processStr)
	}
	if !strings.Contains(processStr, "myproject process") {
		t.Errorf("process file missing header: %s", processStr)
	}
	if !strings.Contains(processStr, ".. rubric:: Usage") {
		t.Errorf("process file missing usage section: %s", processStr)
	}
	if !strings.Contains(processStr, ".. rubric:: Examples") {
		t.Errorf("process file missing examples section: %s", processStr)
	}
	if !strings.Contains(processStr, ".. rubric:: Flags") {
		t.Errorf("process file missing flags section: %s", processStr)
	}
	if !strings.Contains(processStr, "--format") {
		t.Errorf("process file missing format flag: %s", processStr)
	}
	if !strings.Contains(processStr, "--verbose") {
		t.Errorf("process file missing verbose flag: %s", processStr)
	}
	if !strings.Contains(processStr, ".. code-block:: console") {
		t.Errorf("process file missing console code blocks: %s", processStr)
	}
}

func TestMarkdownTemplates(t *testing.T) {
	tempDir := t.TempDir()

	rootCmd := &cobra.Command{
		Use:   "myproject",
		Short: "A sample CLI tool",
		Long:  "myproject is a sample CLI tool for testing Markdown templates",
	}

	subCmd := &cobra.Command{
		Use:   "process [file]",
		Short: "Process a file",
		Long:  "Process a file and perform various transformations on it",
		Example: `Basic usage:
$ myproject process input.txt

With options:
$ myproject process --format json input.txt`,
		Run: func(cmd *cobra.Command, args []string) {},
	}
	subCmd.Flags().String("format", "text", "Output format")
	subCmd.Flags().Bool("verbose", false, "Enable verbose output")

	rootCmd.AddCommand(subCmd)

	indexTemplate, err := os.ReadFile("examples/cli.md")
	if err != nil {
		t.Fatalf("failed to read index template: %v", err)
	}
	commandTemplate, err := os.ReadFile("examples/command.md")
	if err != nil {
		t.Fatalf("failed to read command template: %v", err)
	}

	templates := TemplateInfo{
		IndexFileName:         "index.md",
		IndexTemplate:         string(indexTemplate),
		SingleCommandTemplate: string(commandTemplate),
	}

	err = GenMarkdownTree(rootCmd, tempDir, templates,
		func(s string) string { return "" })
	if err != nil {
		t.Fatalf("GenMarkdownTree failed: %v", err)
	}

	indexPath := filepath.Join(tempDir, "index.md")
	fileExists(t, indexPath)
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index file: %v", err)
	}

	indexStr := string(indexContent)
	if !strings.Contains(indexStr, "# myproject (CLI)") {
		t.Errorf("index missing header: %s", indexStr)
	}
	if !strings.Contains(indexStr, "myproject-process.md") {
		t.Errorf("index missing process command reference: %s", indexStr)
	}

	processPath := filepath.Join(tempDir, "myproject-process.md")
	fileExists(t, processPath)
	processContent, err := os.ReadFile(processPath)
	if err != nil {
		t.Fatalf("failed to read process file: %v", err)
	}

	processStr := string(processContent)
	if !strings.Contains(processStr, "# myproject process") {
		t.Errorf("process file missing header: %s", processStr)
	}
	if !strings.Contains(processStr, "## Usage") {
		t.Errorf("process file missing usage section: %s", processStr)
	}
	if !strings.Contains(processStr, "## Examples") {
		t.Errorf("process file missing examples section: %s", processStr)
	}
	if !strings.Contains(processStr, "## Flags") {
		t.Errorf("process file missing flags section: %s", processStr)
	}
	if !strings.Contains(processStr, "**--format**") {
		t.Errorf("process file missing format flag: %s", processStr)
	}
	if !strings.Contains(processStr, "**--verbose**") {
		t.Errorf("process file missing verbose flag: %s", processStr)
	}
	if !strings.Contains(processStr, "```console") {
		t.Errorf("process file missing console code blocks: %s", processStr)
	}
}

func TestExampleParserWithDollarPrefix(t *testing.T) {
	parser := ExampleParser{
		CommandPrefixes: []string{"$"},
		MinIndent:       2,
	}

	example := `Basic usage:
$ mycommand arg1

Advanced usage:
$ mycommand --flag arg2`

	results := parser.Parse(example)

	if len(results) != 2 {
		t.Fatalf("expected 2 examples, got %d", len(results))
	}

	if results[0].Info != "Basic usage:" {
		t.Errorf("expected info 'Basic usage:', got '%s'", results[0].Info)
	}
	if results[0].Usage != "$ mycommand arg1" {
		t.Errorf("expected usage '$ mycommand arg1', got '%s'", results[0].Usage)
	}

	if results[1].Info != "Advanced usage:" {
		t.Errorf("expected info 'Advanced usage:', got '%s'", results[1].Info)
	}
	if results[1].Usage != "$ mycommand --flag arg2" {
		t.Errorf("expected usage '$ mycommand --flag arg2', got '%s'", results[1].Usage)
	}
}

func TestExampleParserWithPowerShellPrefix(t *testing.T) {
	parser := ExampleParser{
		CommandPrefixes: []string{">"},
		MinIndent:       2,
	}

	example := `Windows example:
> mycommand.exe /arg`

	results := parser.Parse(example)

	if len(results) != 1 {
		t.Fatalf("expected 1 example, got %d", len(results))
	}

	if results[0].Info != "Windows example:" {
		t.Errorf("expected info 'Windows example:', got '%s'", results[0].Info)
	}
	if results[0].Usage != "> mycommand.exe /arg" {
		t.Errorf("expected usage '> mycommand.exe /arg', got '%s'", results[0].Usage)
	}
}

func TestExampleParserWithRootPrefix(t *testing.T) {
	parser := ExampleParser{
		CommandPrefixes: []string{"#"},
		MinIndent:       2,
	}

	example := `Run as root:
# systemctl restart service`

	results := parser.Parse(example)

	if len(results) != 1 {
		t.Fatalf("expected 1 example, got %d", len(results))
	}

	if results[0].Info != "Run as root:" {
		t.Errorf("expected info 'Run as root:', got '%s'", results[0].Info)
	}
	if results[0].Usage != "# systemctl restart service" {
		t.Errorf("expected usage '# systemctl restart service', got '%s'", results[0].Usage)
	}
}

func TestExampleParserWithIndentation(t *testing.T) {
	parser := ExampleParser{
		CommandPrefixes: []string{},
		MinIndent:       2,
	}

	example := `Indented example:
  mycommand indented`

	results := parser.Parse(example)

	if len(results) != 1 {
		t.Fatalf("expected 1 example, got %d", len(results))
	}

	if results[0].Info != "Indented example:" {
		t.Errorf("expected info 'Indented example:', got '%s'", results[0].Info)
	}
	if results[0].Usage != "  mycommand indented" {
		t.Errorf("expected usage '  mycommand indented', got '%s'", results[0].Usage)
	}
}

func TestExampleParserMultiplePrefixes(t *testing.T) {
	parser := ExampleParser{
		CommandPrefixes: []string{"$", ">", "#"},
		MinIndent:       2,
	}

	example := `Unix:
$ mycommand unix

Windows:
> mycommand.exe windows

Root:
# mycommand root`

	results := parser.Parse(example)

	if len(results) != 3 {
		t.Fatalf("expected 3 examples, got %d", len(results))
	}

	if results[0].Info != "Unix:" {
		t.Errorf("expected info 'Unix:', got '%s'", results[0].Info)
	}
	if results[1].Info != "Windows:" {
		t.Errorf("expected info 'Windows:', got '%s'", results[1].Info)
	}
	if results[2].Info != "Root:" {
		t.Errorf("expected info 'Root:', got '%s'", results[2].Info)
	}
}

func TestExampleParserNoInfo(t *testing.T) {
	parser := ExampleParser{
		CommandPrefixes: []string{"$"},
		MinIndent:       2,
	}

	example := `$ mycommand only

$ another command`

	results := parser.Parse(example)

	if len(results) != 2 {
		t.Fatalf("expected 2 examples, got %d", len(results))
	}

	if results[0].Info != "" {
		t.Errorf("expected empty info, got '%s'", results[0].Info)
	}
	if results[0].Usage != "$ mycommand only" {
		t.Errorf("expected usage '$ mycommand only', got '%s'", results[0].Usage)
	}
}

func TestExampleParserMultilineUsage(t *testing.T) {
	parser := ExampleParser{
		CommandPrefixes: []string{"$"},
		MinIndent:       2,
	}

	example := `Complex example:
$ mycommand \
  --flag1 value1 \
  --flag2 value2`

	results := parser.Parse(example)

	if len(results) != 1 {
		t.Fatalf("expected 1 example, got %d", len(results))
	}

	if results[0].Info != "Complex example:" {
		t.Errorf("expected info 'Complex example:', got '%s'", results[0].Info)
	}

	expectedUsage := "$ mycommand \\\n  --flag1 value1 \\\n  --flag2 value2"
	if results[0].Usage != expectedUsage {
		t.Errorf("expected usage '%s', got '%s'", expectedUsage, results[0].Usage)
	}
}

func TestExampleParserEmptyString(t *testing.T) {
	parser := ExampleParser{
		CommandPrefixes: []string{"$"},
		MinIndent:       2,
	}

	results := parser.Parse("")

	if results != nil {
		t.Errorf("expected nil for empty string, got %v", results)
	}
}

func TestExampleParserDefaultPrefixes(t *testing.T) {
	parser := ExampleParser{} // Use defaults

	example := `With dollar:
$ cmd1

With angle:
> cmd2

With hash:
# cmd3`

	results := parser.Parse(example)

	if len(results) != 3 {
		t.Fatalf("expected 3 examples, got %d", len(results))
	}
}

func TestExampleParserNoMatchFallback(t *testing.T) {
	parser := ExampleParser{
		CommandPrefixes: []string{"$"},
		MinIndent:       10, // Very high, won't match normal indents
	}

	example := `mycommand without prefix`

	results := parser.Parse(example)

	if len(results) != 1 {
		t.Fatalf("expected 1 example (fallback), got %d", len(results))
	}

	if results[0].Info != "" {
		t.Errorf("expected empty info for fallback, got '%s'", results[0].Info)
	}
	if results[0].Usage != "mycommand without prefix" {
		t.Errorf("expected full text as usage, got '%s'", results[0].Usage)
	}
}

func TestExampleParserMultilineInfo(t *testing.T) {
	parser := ExampleParser{
		CommandPrefixes: []string{"$"},
		MinIndent:       2,
	}

	example := `This is a longer description
that spans multiple lines
and explains the example:
$ mycommand arg`

	results := parser.Parse(example)

	if len(results) != 1 {
		t.Fatalf("expected 1 example, got %d", len(results))
	}

	expectedInfo := "This is a longer description\nthat spans multiple lines\nand explains the example:"
	if results[0].Info != expectedInfo {
		t.Errorf("expected info '%s', got '%s'", expectedInfo, results[0].Info)
	}
}

func TestGenDocsConcurrent(t *testing.T) {
	template := `{{ .CommandName }}|{{ .Short }}|{{ .Long }}{{ range .Flags }}|{{ .Name }}{{ end }}`

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Run multiple GenDocs calls concurrently
	// Each goroutine creates its own command instance to avoid sharing state
	for i := range goroutines {
		go func(iteration int) {
			defer wg.Done()

			// Create fresh command instance for each goroutine
			// (Cobra commands are not safe to share across goroutines)
			cmd := &cobra.Command{
				Use:   "test",
				Short: "Test command",
				Long:  "A longer description for testing",
				Example: `Example usage:
$ test --flag value`,
			}
			cmd.Flags().String("output", "stdout", "Output destination")

			var buf bytes.Buffer
			if err := GenDocs(cmd, &buf, template); err != nil {
				t.Errorf("concurrent GenDocs iteration %d failed: %v", iteration, err)
				return
			}
			// Verify output contains expected content (help flag is added by Cobra)
			result := buf.String()
			if !strings.Contains(result, "test|Test command|A longer description for testing") {
				t.Errorf("iteration %d: output missing expected content: %s", iteration, result)
			}
			if !strings.Contains(result, "output") {
				t.Errorf("iteration %d: output missing 'output' flag: %s", iteration, result)
			}
		}(i)
	}

	wg.Wait()
}

func TestGenDocsTreeConcurrent(t *testing.T) {
	tempDir := t.TempDir()

	templates := TemplateInfo{
		IndexFileName:         "index.rst",
		IndexTemplate:         `{{ range .Files }}{{ . }}\n{{ end }}`,
		SingleCommandTemplate: `{{ .CommandName }} - {{ .Short }}`,
	}

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Run multiple GenRSTTree calls concurrently to different directories
	// Each goroutine creates its own command tree to avoid sharing state
	for i := range goroutines {
		go func(iteration int) {
			defer wg.Done()

			// Create fresh command instances for each goroutine
			// (Cobra commands are not safe to share across goroutines)
			rootCmd := &cobra.Command{Use: "app"}
			subCmd1 := &cobra.Command{
				Use:   "sub1",
				Short: "Subcommand 1",
				Run:   func(cmd *cobra.Command, args []string) {},
			}
			subCmd2 := &cobra.Command{
				Use:   "sub2",
				Short: "Subcommand 2",
				Run:   func(cmd *cobra.Command, args []string) {},
			}
			rootCmd.AddCommand(subCmd1, subCmd2)

			// Each goroutine uses its own subdirectory
			outputDir := filepath.Join(tempDir, fmt.Sprintf("docs-%d", iteration))
			err := GenRSTTree(rootCmd, outputDir, templates,
				func(s string) string { return "" })
			if err != nil {
				t.Errorf("concurrent GenRSTTree iteration %d failed: %v", iteration, err)
				return
			}

			// Verify files were created
			indexPath := filepath.Join(outputDir, "index.rst")
			if _, err := os.Stat(indexPath); err != nil {
				t.Errorf("iteration %d: index file not created: %v", iteration, err)
			}
		}(i)
	}

	wg.Wait()
}

// --- Options, extended data, helpers, parser configuration, validation ---

func newFlagTestCommand() (*cobra.Command, *cobra.Command) {
	rootCmd := &cobra.Command{Use: "root", Short: "Root"}
	rootCmd.PersistentFlags().StringP("config", "c", "", "Config file")

	subCmd := &cobra.Command{
		Use:   "sub",
		Short: "Sub command",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
	subCmd.Flags().StringP("name", "n", "default", "Name to use")
	subCmd.Flags().Bool("force", false, "Force it")
	subCmd.Flags().String("secret", "", "Hidden flag")
	subCmd.Flags().String("old", "", "Old flag")
	subCmd.Flags().String("lazy", "", "Optional value")
	subCmd.Flags().Lookup("lazy").NoOptDefVal = "auto"
	_ = subCmd.Flags().MarkHidden("secret")
	_ = subCmd.Flags().MarkDeprecated("old", "use --name")
	_ = subCmd.MarkFlagRequired("force")
	rootCmd.AddCommand(subCmd)
	return rootCmd, subCmd
}

func flagsByName(flags []FlagInfo) map[string]FlagInfo {
	m := map[string]FlagInfo{}
	for _, f := range flags {
		m[f.Name] = f
	}
	return m
}

func TestGenDocsFlagInfoFields(t *testing.T) {
	_, subCmd := newFlagTestCommand()
	cfg := newConfig(nil)
	data := cfg.commandData(subCmd)
	flags := flagsByName(data.Flags)

	if _, ok := flags["secret"]; ok {
		t.Errorf("hidden flag should be skipped by default: %+v", data.Flags)
	}
	name := flags["name"]
	if name.Shorthand != "n" || name.Type != "string" || name.DefaultValue != "default" {
		t.Errorf("unexpected name flag: %+v", name)
	}
	if !flags["force"].Required || flags["force"].Type != "bool" {
		t.Errorf("force should be required bool: %+v", flags["force"])
	}
	if _, ok := flags["old"]; ok {
		t.Errorf("deprecated flags are hidden by pflag and must be skipped by default: %+v", data.Flags)
	}
	if flags["lazy"].NoOptDefVal != "auto" {
		t.Errorf("NoOptDefVal not captured: %+v", flags["lazy"])
	}
	help := flags["help"]
	if !help.SetByCobra || help.Shorthand != "h" {
		t.Errorf("help flag should be marked SetByCobra with shorthand h: %+v", help)
	}
	if _, ok := flags["config"]; ok {
		t.Errorf("inherited flag must not be in Flags: %+v", data.Flags)
	}
	inherited := flagsByName(data.InheritedFlags)
	if inherited["config"].Shorthand != "c" {
		t.Errorf("inherited config flag missing or wrong: %+v", data.InheritedFlags)
	}
}

func TestGenDocsWithHiddenFlags(t *testing.T) {
	_, subCmd := newFlagTestCommand()
	cfg := newConfig([]Option{WithHiddenFlags()})
	flags := flagsByName(cfg.commandData(subCmd).Flags)
	secret, ok := flags["secret"]
	if !ok || !secret.Hidden {
		t.Errorf("hidden flag should be included and marked Hidden: %+v", flags)
	}
	old, ok := flags["old"]
	if !ok || old.Deprecated != "use --name" || !old.Hidden {
		t.Errorf("deprecated flag should be included with its message: %+v", flags)
	}
}

func TestGenDocsWithoutHelpFlag(t *testing.T) {
	_, subCmd := newFlagTestCommand()
	cfg := newConfig([]Option{WithoutHelpFlag()})
	flags := flagsByName(cfg.commandData(subCmd).Flags)
	if _, ok := flags["help"]; ok {
		t.Errorf("help flag should be omitted: %+v", flags)
	}
	if _, ok := flags["name"]; !ok {
		t.Errorf("regular flags must remain: %+v", flags)
	}
}

func TestGenDocsHelpFlagInheritedWhenRootDefinesIt(t *testing.T) {
	rootCmd := &cobra.Command{Use: "root"}
	rootCmd.PersistentFlags().BoolP("help", "h", false, "Print help")
	subCmd := &cobra.Command{Use: "sub", Run: func(cmd *cobra.Command, args []string) {}}
	rootCmd.AddCommand(subCmd)

	data := newConfig(nil).commandData(subCmd)
	if _, ok := flagsByName(data.Flags)["help"]; ok {
		t.Errorf("application-defined persistent help must not be a local flag: %+v", data.Flags)
	}
	help, ok := flagsByName(data.InheritedFlags)["help"]
	if !ok || help.SetByCobra {
		t.Errorf("application-defined help must be inherited and not SetByCobra: %+v", data.InheritedFlags)
	}
}

func TestGenDocsCommandDataFields(t *testing.T) {
	rootCmd := &cobra.Command{Use: "app"}
	groupCmd := &cobra.Command{
		Use:         "group",
		Short:       "Group",
		Aliases:     []string{"g", "grp"},
		Deprecated:  "use app other",
		Annotations: map[string]string{"related": "app other", "custom": "value"},
	}
	leafCmd := &cobra.Command{Use: "leaf", Short: "Leaf", Run: func(cmd *cobra.Command, args []string) {}}
	hiddenCmd := &cobra.Command{Use: "hidden", Hidden: true, Run: func(cmd *cobra.Command, args []string) {}}
	topicCmd := &cobra.Command{Use: "topic", Short: "Help topic"}
	groupCmd.AddCommand(leafCmd, hiddenCmd, topicCmd)
	rootCmd.AddCommand(groupCmd)

	cfg := newConfig(nil)
	cfg.ext = ".md"
	data := cfg.commandData(groupCmd)

	if data.Parent != "app" || data.File != "app-group.md" || data.Ref != "ref_app_group" {
		t.Errorf("unexpected identity fields: %+v", data)
	}
	if data.Runnable {
		t.Errorf("group has no Run and must not be Runnable")
	}
	if data.Deprecated != "use app other" || len(data.Aliases) != 2 {
		t.Errorf("Deprecated/Aliases not captured: %+v", data)
	}
	if data.Annotations["custom"] != "value" {
		t.Errorf("Annotations not exposed: %+v", data.Annotations)
	}
	if len(data.Subcommands) != 1 || data.Subcommands[0].Name != "app group leaf" ||
		data.Subcommands[0].File != "app-group-leaf.md" || data.Subcommands[0].Ref != "ref_app_group_leaf" {
		t.Errorf("Subcommands should list only the documented leaf: %+v", data.Subcommands)
	}
	if data.HeadingLen != len("app group") {
		t.Errorf("HeadingLen = %d", data.HeadingLen)
	}
}

func TestGenDocsHeadingLenUnicode(t *testing.T) {
	cmd := &cobra.Command{Use: "café"}
	data := newConfig(nil).commandData(cmd)
	if data.HeadingLen != 4 {
		t.Errorf("HeadingLen should count runes, got %d", data.HeadingLen)
	}
}

func TestGenDocsWithFuncs(t *testing.T) {
	cmd := &cobra.Command{Use: "app", Short: "hello"}
	var out bytes.Buffer
	err := GenDocs(cmd, &out, `{{ .Short | shout }} {{ .Short | upper }}`,
		WithFuncs(template.FuncMap{
			"shout": func(s string) string { return s + "!" },
			"upper": func(s string) string { return "overridden" },
		}))
	if err != nil {
		t.Fatalf("GenDocs failed: %v", err)
	}
	if out.String() != "hello! overridden" {
		t.Errorf("unexpected output %q", out.String())
	}
}

func TestGenDocsUnknownFuncFailsAtParse(t *testing.T) {
	cmd := &cobra.Command{Use: "app"}
	err := GenDocs(cmd, io.Discard, `{{ .Short | nosuch }}`)
	if err == nil || !strings.Contains(err.Error(), "nosuch") {
		t.Errorf("expected parse error naming the function, got %v", err)
	}
}

func TestGenDocsWithExampleParser(t *testing.T) {
	cmd := &cobra.Command{
		Use:     "app",
		Example: "Run it:\nPS> app run\n\nOther:\n  indented but not a command",
	}
	var out bytes.Buffer
	tmpl := `{{ range .Examples }}[{{ .Info }}|{{ .Usage }}]{{ end }}`
	err := GenDocs(cmd, &out, tmpl, WithExampleParser(ExampleParser{
		CommandPrefixes:        []string{"PS>"},
		DisableIndentDetection: true,
	}))
	if err != nil {
		t.Fatalf("GenDocs failed: %v", err)
	}
	want := "[Run it:|PS> app run][|Other:\n  indented but not a command]"
	if out.String() != want {
		t.Errorf("got %q, want %q", out.String(), want)
	}
}

func TestTemplateHelpers(t *testing.T) {
	funcs := builtinFuncs()
	cases := []struct {
		tmpl string
		want string
	}{
		{`{{ slug "My App  sub-cmd!" }}`, "my-app-sub-cmd"},
		{`{{ anchor "ref_my app" }}`, "ref_my-app"},
		{`{{ titleCase "list all iTems" }}`, "List All ITems"},
		{`{{ "app sub" | trimPrefix "app " }}`, "sub"},
		{`{{ "Short." | trimSuffix "." }}`, "Short"},
		{`{{ "  x  " | trimSpace }}`, "x"},
		{`{{ "Ab" | lower }}{{ "Ab" | upper }}`, "abAB"},
		{`{{ .Items | join ", " }}`, "a, b"},
		{`{{ "a b" | replace " " "_" }}`, "a_b"},
		{`{{ "a b" | replaceSpaces }}`, "a_b"},
		{`{{ repeat "-" 3 }}`, "---"},
		{`{{ "x\ny" | indent 2 }}`, "  x\n  y"},
	}
	for _, tc := range cases {
		tmpl, err := template.New("t").Funcs(funcs).Parse(tc.tmpl)
		if err != nil {
			t.Fatalf("%s: parse error: %v", tc.tmpl, err)
		}
		var out bytes.Buffer
		if err := tmpl.Execute(&out, struct{ Items []string }{Items: []string{"a", "b"}}); err != nil {
			t.Fatalf("%s: exec error: %v", tc.tmpl, err)
		}
		if out.String() != tc.want {
			t.Errorf("%s: got %q, want %q", tc.tmpl, out.String(), tc.want)
		}
	}
}

func TestIndexTemplateHasCommandsAndFuncs(t *testing.T) {
	tempDir := t.TempDir()
	rootCmd := &cobra.Command{Use: "app"}
	rootCmd.AddCommand(&cobra.Command{Use: "one", Short: "First", Run: func(cmd *cobra.Command, args []string) {}})
	rootCmd.AddCommand(&cobra.Command{Use: "two", Short: "Second", Run: func(cmd *cobra.Command, args []string) {}})

	templates := TemplateInfo{
		IndexFileName:         "index.md",
		IndexTemplate:         `{{ .Root }}:{{ range .Commands }} [{{ .Name | slug }}]({{ .File }}) {{ .Short }};{{ end }}`,
		SingleCommandTemplate: `{{ .CommandName }}`,
	}
	if err := GenMarkdownTree(rootCmd, tempDir, templates, nil); err != nil {
		t.Fatalf("GenMarkdownTree failed: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(tempDir, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "app: [app-one](app-one.md) First; [app-two](app-two.md) Second;"
	if string(content) != want {
		t.Errorf("got %q, want %q", content, want)
	}
}

func TestGenDocsTreeDryRunWritesNothing(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "out")
	rootCmd := &cobra.Command{Use: "app"}
	rootCmd.AddCommand(&cobra.Command{Use: "one", Run: func(cmd *cobra.Command, args []string) {}})

	templates := TemplateInfo{IndexFileName: "index.rst", IndexTemplate: "i", SingleCommandTemplate: "c"}
	if err := GenRSTTree(rootCmd, outDir, templates, nil, WithDryRun()); err != nil {
		t.Fatalf("dry run failed: %v", err)
	}
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Errorf("dry run must not create the output directory (stat err: %v)", err)
	}

	// Errors are still reported in dry-run mode.
	bad := TemplateInfo{IndexFileName: "index.rst", IndexTemplate: "i", SingleCommandTemplate: "{{ .Nope }}"}
	if err := GenRSTTree(rootCmd, outDir, bad, nil, WithDryRun()); err == nil {
		t.Errorf("dry run should surface template errors")
	}
}

func TestGenDocsTreeTemplateErrorLeavesNoPartialFile(t *testing.T) {
	tempDir := t.TempDir()
	rootCmd := &cobra.Command{Use: "app"}
	rootCmd.AddCommand(&cobra.Command{Use: "one", Run: func(cmd *cobra.Command, args []string) {}})

	templates := TemplateInfo{
		IndexFileName:         "index.md",
		IndexTemplate:         "i",
		SingleCommandTemplate: "header {{ .Missing }}",
	}
	err := GenMarkdownTree(rootCmd, tempDir, templates, func(string) string { return "prepended" })
	if err == nil {
		t.Fatal("expected an error")
	}
	if names := readDirNames(t, tempDir); len(names) != 0 {
		t.Errorf("no files should be written on error, found %v", names)
	}
}

func TestGenDocsTreeNilFilePrepender(t *testing.T) {
	tempDir := t.TempDir()
	rootCmd := &cobra.Command{Use: "app"}
	rootCmd.AddCommand(&cobra.Command{Use: "one", Run: func(cmd *cobra.Command, args []string) {}})
	templates := TemplateInfo{IndexFileName: "index.md", IndexTemplate: "i", SingleCommandTemplate: "c"}
	if err := GenMarkdownTree(rootCmd, tempDir, templates, nil); err != nil {
		t.Fatalf("nil filePrepender should be accepted: %v", err)
	}
	fileExists(t, filepath.Join(tempDir, "app-one.md"))
}

func TestGenDocsTreeInvalidCommandTemplateFailsBeforeWriting(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "out")
	rootCmd := &cobra.Command{Use: "app"}
	rootCmd.AddCommand(&cobra.Command{Use: "one", Run: func(cmd *cobra.Command, args []string) {}})
	templates := TemplateInfo{IndexFileName: "index.md", IndexTemplate: "i", SingleCommandTemplate: "{{ if }}"}
	if err := GenMarkdownTree(rootCmd, outDir, templates, nil); err == nil {
		t.Fatal("expected parse error")
	}
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Errorf("output directory must not be created when templates fail to parse")
	}
}

func TestExampleParserCRLF(t *testing.T) {
	parser := ExampleParser{}
	examples := parser.Parse("Info:\r\n  $ cmd one\r\n\r\nMore:\r\n  $ cmd two")
	if len(examples) != 2 {
		t.Fatalf("expected 2 examples, got %d: %+v", len(examples), examples)
	}
	if examples[0].Info != "Info:" || examples[0].Usage != "  $ cmd one" {
		t.Errorf("unexpected first example: %+v", examples[0])
	}
}

func TestExampleParserTabIndent(t *testing.T) {
	parser := ExampleParser{CommandPrefixes: []string{"%"}}
	examples := parser.Parse("Info:\n\tcmd one")
	if len(examples) != 1 || examples[0].Info != "Info:" || examples[0].Usage != "\tcmd one" {
		t.Errorf("tab-indented line should be a command: %+v", examples)
	}
}

func TestExampleParserCustomBlockSeparator(t *testing.T) {
	parser := ExampleParser{BlockSeparator: "\n---\n"}
	examples := parser.Parse("A:\n  $ a\n\nstill a\n---\nB:\n  $ b")
	if len(examples) != 2 {
		t.Fatalf("expected 2 examples, got %d: %+v", len(examples), examples)
	}
	if examples[0].Usage != "  $ a\n\nstill a" || examples[1].Info != "B:" {
		t.Errorf("unexpected split: %+v", examples)
	}
}

func TestExampleParserDisableIndentDetection(t *testing.T) {
	parser := ExampleParser{DisableIndentDetection: true}
	examples := parser.Parse("Info:\n  not a command\n  $ cmd")
	if len(examples) != 1 {
		t.Fatalf("expected 1 example, got %+v", examples)
	}
	if examples[0].Info != "Info:\n  not a command" || examples[0].Usage != "  $ cmd" {
		t.Errorf("indented prose must stay in Info when indent detection is off: %+v", examples[0])
	}
}

func TestExampleParserMinIndentZeroUsesDefault(t *testing.T) {
	parser := ExampleParser{MinIndent: 0, CommandPrefixes: []string{"%"}}
	examples := parser.Parse("Info:\n  cmd")
	if len(examples) != 1 || examples[0].Usage != "  cmd" {
		t.Errorf("MinIndent 0 should fall back to the default of 2: %+v", examples)
	}
}

func TestValidateTemplates(t *testing.T) {
	good := TemplateInfo{
		IndexFileName: "index.md",
		IndexTemplate: `{{ .Root }}{{ range .Commands }}{{ .Name | slug }}{{ end }}{{ range .Files }}{{ . }}{{ end }}`,
		SingleCommandTemplate: `{{ .CommandName }}{{ if .Deprecated }}D{{ else }}-{{ end }}` +
			`{{ range .Flags }}{{ .Shorthand }}{{ end }}{{ range .Subcommands }}{{ .Ref }}{{ end }}{{ index .Annotations "related" }}`,
	}
	if err := ValidateTemplates(good); err != nil {
		t.Errorf("valid templates rejected: %v", err)
	}

	cases := map[string]TemplateInfo{
		"unknown field in command": {IndexFileName: "i", IndexTemplate: "i", SingleCommandTemplate: "{{ .Nope }}"},
		"unknown field in index":   {IndexFileName: "i", IndexTemplate: "{{ .Nope }}", SingleCommandTemplate: "c"},
		"unknown func":             {IndexFileName: "i", IndexTemplate: "i", SingleCommandTemplate: "{{ nosuch . }}"},
		"else branch":              {IndexFileName: "i", IndexTemplate: "i", SingleCommandTemplate: "{{ if .Deprecated }}ok{{ else }}{{ .Nope }}{{ end }}"},
		"empty":                    {},
	}
	for name, tc := range cases {
		if err := ValidateTemplates(tc); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}

	withFunc := TemplateInfo{IndexFileName: "i", IndexTemplate: "i", SingleCommandTemplate: "{{ custom .Short }}"}
	if err := ValidateTemplates(withFunc, WithFuncs(template.FuncMap{"custom": strings.ToUpper})); err != nil {
		t.Errorf("WithFuncs should be honoured: %v", err)
	}
}

func TestValidateTemplatesAcceptsExamples(t *testing.T) {
	for _, pair := range [][2]string{{"cli.md", "command.md"}, {"cli.rst", "command.rst"}} {
		index, err := os.ReadFile(filepath.Join("examples", pair[0]))
		if err != nil {
			t.Fatal(err)
		}
		command, err := os.ReadFile(filepath.Join("examples", pair[1]))
		if err != nil {
			t.Fatal(err)
		}
		templates := TemplateInfo{IndexFileName: pair[0], IndexTemplate: string(index), SingleCommandTemplate: string(command)}
		if err := ValidateTemplates(templates); err != nil {
			t.Errorf("%v: %v", pair, err)
		}
	}
}
