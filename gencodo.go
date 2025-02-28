package gencodo

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type FlagDetail struct {
	Name         string
	Usage        string
	DefaultValue string
}

type ExampleDetail struct {
	Info  string
	Usage string
}

type TemplateDetail struct {
	IndexFileName         string
	IndexTemplate         string
	SingleCommandTemplate string
}

func GenReSTCustom(cmd *cobra.Command, w io.Writer, templateContent string, linkHandler func(string, string) string) error {
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

	entries := strings.Split(cmd.Example, "\n\n")
	var structuredExamples []ExampleDetail

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		lines := strings.Split(entry, "\n")
		var infoLines, usageLines []string

		for i, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "$") {
				infoLines = lines[:i]
				usageLines = lines[i:]
				break
			}
		}

		if len(infoLines) > 0 && len(usageLines) > 0 {
			structuredExamples = append(structuredExamples, ExampleDetail{
				Info:  strings.Join(infoLines, "\n"),
				Usage: strings.Join(usageLines, "\n"),
			})
		}
	}

	flags := cmd.NonInheritedFlags()
	var flagDetails []FlagDetail
	flags.VisitAll(func(flag *pflag.Flag) {
		flagDetails = append(flagDetails, FlagDetail{
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
		Examples        []ExampleDetail
		Flags           []FlagDetail
		HeadingLen      int
		RelatedCommands []string
	}{
		Ref:             ref,
		CommandName:     name,
		Short:           short,
		Long:            long,
		Synopsis:        cmd.UseLine(),
		Examples:        structuredExamples,
		Flags:           flagDetails,
		HeadingLen:      headinglen,
		RelatedCommands: relatedCommands,
	}

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

func GenReSTTreeCustom(
	cmd *cobra.Command,
	dir string,
	templates TemplateDetail,
	filePrepender func(string) string,
	linkHandler func(string, string) string,
) error {
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
		if err := GenReSTCustom(c, f, templates.SingleCommandTemplate, linkHandler); err != nil {
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
