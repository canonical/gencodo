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
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	err := GenDocs(cmd, &output, templateContent, func(cmdPath, _ string) string { return cmdPath })
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
	err := GenDocs(cmd, &output, templateContent, func(cmdPath, _ string) string { return cmdPath })
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
	err := GenDocs(cmd, &output, templateContent, func(cmdPath, _ string) string { return cmdPath })
	if err != nil {
		t.Fatalf("GenDocs failed: %v", err)
	}
	if !strings.Contains(output.String(), "  indentrepeatXXX") {
		t.Errorf("unexpected template output: %s", output.String())
	}
}

func TestGenDocsTree(t *testing.T) {
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
		IndexFileName:         "root.rst",
		IndexTemplate:         `{{ range .Files }}{{ . }}\n{{ end }}`,
		SingleCommandTemplate: `{{ .CommandName }} - {{ .Short }}`,
	}
	err := GenDocsTree(rootCmd, tempDir, templates, func(s string) string { return "" }, func(cmdPath, _ string) string { return cmdPath })
	if err != nil {
		t.Fatalf("GenDocsTree failed: %v", err)
	}
	fileExists(t, filepath.Join(tempDir, "root.rst"))
	fileExists(t, filepath.Join(tempDir, "root-sub1.rst"))
	fileExists(t, filepath.Join(tempDir, "root-sub2.rst"))
}

func TestGenDocsTreeNested(t *testing.T) {
	tempDir := t.TempDir()
	rootCmd := &cobra.Command{Use: "top"}
	subCmdA := &cobra.Command{Use: "A", Short: "A", Run: func(cmd *cobra.Command, args []string) {}}
	subCmdB := &cobra.Command{Use: "B", Short: "B", Run: func(cmd *cobra.Command, args []string) {}}
	subCmdC := &cobra.Command{Use: "C", Short: "C", Run: func(cmd *cobra.Command, args []string) {}}
	subCmdA1 := &cobra.Command{Use: "A1", Short: "A1", Run: func(cmd *cobra.Command, args []string) {}}
	subCmdA.AddCommand(subCmdA1)
	rootCmd.AddCommand(subCmdA, subCmdB, subCmdC)
	templates := TemplateInfo{
		IndexFileName:         "top.rst",
		IndexTemplate:         `{{ range .Files }}{{ . }}\n{{ end }}`,
		SingleCommandTemplate: `{{ .CommandName }} - {{ .Short }}`,
	}
	err := GenDocsTree(rootCmd, tempDir, templates, func(s string) string { return "" }, func(cmdPath, _ string) string { return cmdPath })
	if err != nil {
		t.Fatalf("GenDocsTree failed: %v", err)
	}
	names := readDirNames(t, tempDir)
	if !containsAll(names, []string{"top.rst", "top-A.rst", "top-A-A1.rst", "top-B.rst", "top-C.rst"}) {
		t.Errorf("unexpected files: %v", names)
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
	err := GenDocs(cmd, &output, templateContent, func(cmdPath, _ string) string { return cmdPath })
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
	err := GenDocs(cmd, &output, templateContent, func(cmdPath, _ string) string { return cmdPath })
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
	err := GenDocs(subCmd, &output, templateContent, func(cmdPath, _ string) string { return cmdPath })
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
	err := GenDocs(cmd, &output, templateContent, func(cmdPath, _ string) string { return cmdPath })
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

	err := GenDocsTree(rootCmd, tempDir, templates, func(s string) string { return headerContent }, func(cmdPath, _ string) string { return cmdPath })
	if err != nil {
		t.Fatalf("GenDocsTree failed: %v", err)
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

	err := GenDocsTree(rootCmd, tempDir, templates, func(s string) string { return "" }, func(cmdPath, _ string) string { return cmdPath })
	if err != nil {
		t.Fatalf("GenDocsTree failed: %v", err)
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
	err := GenDocs(cmd, &output, invalidTemplate, func(cmdPath, _ string) string { return cmdPath })
	if err == nil {
		t.Fatal("expected error for invalid template, got nil")
	}
}

func TestGenDocsTreeInvalidIndexTemplate(t *testing.T) {
	tempDir := t.TempDir()
	rootCmd := &cobra.Command{Use: "root"}
	subCmd := &cobra.Command{Use: "sub", Short: "Sub", Run: func(cmd *cobra.Command, args []string) {}}
	rootCmd.AddCommand(subCmd)

	templates := TemplateInfo{
		IndexFileName:         "index.rst",
		IndexTemplate:         `{{ .Files `,
		SingleCommandTemplate: `{{ .CommandName }}`,
	}

	err := GenDocsTree(rootCmd, tempDir, templates, func(s string) string { return "" }, func(cmdPath, _ string) string { return cmdPath })
	if err == nil {
		t.Fatal("expected error for invalid index template, got nil")
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
	err := GenDocs(cmd, &output, templateContent, func(cmdPath, _ string) string { return cmdPath })
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
	err := GenDocs(cmd, &output, templateContent, func(cmdPath, _ string) string { return cmdPath })
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
	err := GenDocs(cmd, &output, templateContent, func(cmdPath, _ string) string { return cmdPath })
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
	err := GenDocs(subCmd, &output, templateContent, func(cmdPath, _ string) string { return cmdPath })
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
