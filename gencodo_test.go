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

func TestGenDocsCustom(t *testing.T) {
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
	err := GenDocsCustom(cmd, &output, templateContent, func(cmdPath, _ string) string { return cmdPath })
	if err != nil {
		t.Fatalf("GenDocsCustom failed: %v", err)
	}
	expected := "example|An example command|A longer description of the example command.|Foo:|$ example foo|Bar:|$ example bar|foo_bar|bar|baz"
	if output.String() != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, output.String())
	}
}

func TestGenDocsCustomEmptyExample(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "noexample",
		Short: "No example command",
	}
	var output bytes.Buffer
	templateContent := `{{ .CommandName }}|{{ .Short }}|{{ .Long }}|`
	err := GenDocsCustom(cmd, &output, templateContent, func(cmdPath, _ string) string { return cmdPath })
	if err != nil {
		t.Fatalf("GenDocsCustom failed: %v", err)
	}
	if !strings.Contains(output.String(), "noexample|No example command|No example command|") {
		t.Errorf("unexpected output: %s", output.String())
	}
}

func TestGenDocsCustomIndentRepeat(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "indentrepeat",
		Short: "Testing indent and repeat",
	}
	var output bytes.Buffer
	templateContent := `{{ indent 2 .CommandName }}{{ repeat "X" 3 }}`
	err := GenDocsCustom(cmd, &output, templateContent, func(cmdPath, _ string) string { return cmdPath })
	if err != nil {
		t.Fatalf("GenDocsCustom failed: %v", err)
	}
	if !strings.Contains(output.String(), "  indentrepeatXXX") {
		t.Errorf("unexpected template output: %s", output.String())
	}
}

func TestGenDocsTreeCustom(t *testing.T) {
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
	err := GenDocsTreeCustom(rootCmd, tempDir, templates, func(s string) string { return "" }, func(cmdPath, _ string) string { return cmdPath })
	if err != nil {
		t.Fatalf("GenDocsTreeCustom failed: %v", err)
	}
	fileExists(t, filepath.Join(tempDir, "root.rst"))
	fileExists(t, filepath.Join(tempDir, "root-sub1.rst"))
	fileExists(t, filepath.Join(tempDir, "root-sub2.rst"))
}

func TestGenDocsTreeCustomNested(t *testing.T) {
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
	err := GenDocsTreeCustom(rootCmd, tempDir, templates, func(s string) string { return "" }, func(cmdPath, _ string) string { return cmdPath })
	if err != nil {
		t.Fatalf("GenDocsTreeCustom failed: %v", err)
	}
	names := readDirNames(t, tempDir)
	if !containsAll(names, []string{"top.rst", "top-A.rst", "top-A-A1.rst", "top-B.rst", "top-C.rst"}) {
		t.Errorf("unexpected files: %v", names)
	}
}
