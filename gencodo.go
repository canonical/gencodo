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

// Package gencodo generates documentation for Cobra CLI applications from
// user-supplied Go templates.
//
// Gencodo extracts structured data from a [cobra.Command] tree (metadata,
// flags, parsed examples, relationships) and renders it through text/template
// templates, so the output format is entirely up to the caller: Markdown,
// reStructuredText, JSON, YAML, or anything else text-based.
//
// The main entry points are [GenDocs] (one command to an io.Writer),
// [GenMarkdownTree] and [GenRSTTree] (a whole command tree to a directory),
// and [ValidateTemplates] (render templates against synthetic data without
// writing files). Behaviour is tuned with functional [Option] values.
//
// Like Cobra's own doc generators, gencodo calls InitDefaultHelpCmd and
// InitDefaultHelpFlag on every command it documents. This mutates the command
// tree (adds the help command and --help flag), so do not share one tree
// across goroutines; build a fresh tree per goroutine instead.
package gencodo

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FlagInfo holds metadata about a CLI flag extracted from Cobra commands.
//
// [CommandData.Flags] contains non-inherited flags (defined directly on the
// command), [CommandData.InheritedFlags] contains persistent flags declared by
// parent commands. Hidden flags are omitted unless [WithHiddenFlags] is used;
// note that pflag's MarkDeprecated also hides a flag.
//
// Example usage in templates:
//
//	{{- range .Flags }}
//	{{ if .Shorthand }}-{{ .Shorthand }}, {{ end }}--{{ .Name }}    {{ .Usage }}
//	  Default: {{ .DefaultValue }}
//	{{- end }}
//
// The DefaultValue field contains the flag's string representation as shown
// in help text (e.g., "false" for booleans, "[]" for empty slices).
type FlagInfo struct {
	Name                string // Flag name (without leading dashes)
	Shorthand           string // One-letter shorthand, or "" if none
	Usage               string // Description of the flag
	DefaultValue        string // Default value of the flag, as a string
	Type                string // Value type as reported by pflag, e.g. "string", "bool", "stringSlice"
	NoOptDefVal         string // Value used when the flag is given without an argument, if any
	Deprecated          string // Deprecation message, or "" if the flag is not deprecated
	ShorthandDeprecated string // Deprecation message for the shorthand, or ""
	Hidden              bool   // Flag is hidden from help (only present with WithHiddenFlags)
	Required            bool   // Flag was marked required with cobra.MarkFlagRequired
	SetByCobra          bool   // Flag was added automatically by Cobra (the --help flag)
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
//	    {Info: "Display help for this command:", Usage: "  $ myapp help"},
//	    {Info: "List available resources:", Usage: "  $ myapp list"},
//	}
//
// Usage keeps the command lines verbatim, including any prefix and indentation,
// so templates decide how to present them. If no command line is detected, the
// entire block becomes Usage with empty Info.
type ExampleInfo struct {
	Info  string // Description of the example
	Usage string // Example command usage, verbatim
}

// ExampleParser configures how examples are parsed from Cobra command example strings.
// It splits example text into blocks and identifies the first command line of
// each block using prefixes or indentation heuristics. Everything before the
// first command line is the description (Info); the command line and all
// following lines of the block are the usage (Usage).
//
// Default behavior (zero values):
//   - CommandPrefixes: ["$", ">", "#"] (common shell prompts)
//   - MinIndent: 2 (lines indented ≥2 spaces are considered commands)
//   - BlockSeparator: "\n\n" (blocks are separated by a blank line)
//   - DisableIndentDetection: false
//
// The parser first checks for CommandPrefixes. If a line starts with any prefix
// (after trimming leading whitespace), it's treated as a command. The prefix is
// kept in the Usage field.
//
// If no prefix matches, DisableIndentDetection is false, and the line is
// indented by at least MinIndent columns (a tab counts as MinIndent columns),
// the line is considered a command. Indentation is preserved in the Usage field.
// A MinIndent of zero or less selects the default of 2; set
// DisableIndentDetection to rely on prefixes only.
//
// Caveats: a description line that starts with "#" (a shell comment) matches
// the default prefixes, and a description line indented by two or more spaces
// matches the indentation rule; adjust CommandPrefixes, MinIndent, or
// DisableIndentDetection if your examples use such lines. Windows line endings
// (\r\n) are normalised to \n before parsing.
//
// Example - Shell commands only:
//
//	parser := ExampleParser{CommandPrefixes: []string{"$"}, DisableIndentDetection: true}
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
//
// Pass a parser to the generators with [WithExampleParser].
type ExampleParser struct {
	CommandPrefixes        []string // Prefixes that indicate command lines (e.g., "$", ">", "#")
	MinIndent              int      // Minimum indentation (columns) to consider a line as a command
	BlockSeparator         string   // Separator between example blocks; default "\n\n"
	DisableIndentDetection bool     // Detect commands by prefix only, ignoring indentation
}

// DefaultCommandPrefixes are the prefixes used when ExampleParser.CommandPrefixes is empty.
var DefaultCommandPrefixes = []string{"$", ">", "#"}

// DefaultMinIndent is the indentation used when ExampleParser.MinIndent is zero or less.
const DefaultMinIndent = 2

// DefaultBlockSeparator is the block separator used when ExampleParser.BlockSeparator is empty.
const DefaultBlockSeparator = "\n\n"

// TemplateInfo stores Go templates for generating documentation from Cobra commands.
// All fields are required and validated via the Validate() method.
//
// Templates use Go's text/template syntax. IndexTemplate receives an
// [IndexData] value and SingleCommandTemplate receives a [CommandData] value;
// both have access to the built-in helper functions (see the README) and any
// functions added with [WithFuncs].
//
// Example - Simple Markdown template:
//
//	templateInfo := TemplateInfo{
//	    IndexFileName: "index.md",
//	    IndexTemplate: "# Commands\n{{range .Commands}}- [{{.Name}}]({{.File}}): {{.Short}}\n{{end}}",
//	    SingleCommandTemplate: "# {{.CommandName}}\n\n{{.Long}}\n",
//	}
//	if err := templateInfo.Validate(); err != nil {
//	    return err
//	}
//
// Use [ValidateTemplates] to check that templates also render. For complete
// examples, see examples/command.md and examples/command.rst in the repository.
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

// CommandSummary is a short description of a command, used for listing
// subcommands in [CommandData.Subcommands] and all commands in [IndexData.Commands].
type CommandSummary struct {
	Name  string // Full command path, e.g. "myapp sub"
	Short string // Short description
	Ref   string // Reference ID, same scheme as CommandData.Ref
	File  string // File name of the command's documentation (without directory)
}

// CommandData is the value passed to TemplateInfo.SingleCommandTemplate.
type CommandData struct {
	Ref             string            // Reference ID: "ref_" + command path with spaces replaced by underscores
	CommandName     string            // Full command path, e.g. "myapp subcommand"
	File            string            // File name of this command's documentation (without directory)
	Short           string            // Short description
	Long            string            // Long description (falls back to Short if empty)
	Synopsis        string            // Usage line from Cobra, e.g. "myapp sub [flags]"
	Examples        []ExampleInfo     // Parsed examples
	Flags           []FlagInfo        // Non-inherited flags
	InheritedFlags  []FlagInfo        // Persistent flags inherited from parent commands
	HeadingLen      int               // Length of CommandName in characters (for heading underlines)
	RelatedCommands []string          // From Annotations["related"] (comma-separated)
	Aliases         []string          // Command aliases
	Deprecated      string            // Deprecation message, or "" if not deprecated
	Runnable        bool              // Whether the command has a Run function
	Parent          string            // Full path of the parent command, or "" for the root
	Subcommands     []CommandSummary  // Available, non-hidden subcommands
	Annotations     map[string]string // Raw command annotations
}

// IndexData is the value passed to TemplateInfo.IndexTemplate.
type IndexData struct {
	Root     string           // Name of the root command
	Files    []string         // Generated command file names, in generation (Cobra) order
	Commands []CommandSummary // Generated commands, in the same order as Files
}

// Option configures documentation generation. Options are created with the
// With* functions and passed to [GenDocs], [GenMarkdownTree], [GenRSTTree],
// and [ValidateTemplates].
type Option func(*config)

type config struct {
	parser      ExampleParser
	funcs       template.FuncMap
	dryRun      bool
	hiddenFlags bool
	helpFlag    bool
	ext         string
}

func newConfig(opts []Option) *config {
	cfg := &config{helpFlag: true}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	return cfg
}

// WithExampleParser sets the parser used to split command examples into
// [ExampleInfo] values. The default parser uses [DefaultCommandPrefixes] and
// [DefaultMinIndent].
func WithExampleParser(parser ExampleParser) Option {
	return func(c *config) { c.parser = parser }
}

// WithFuncs adds functions to the template function map of both the index and
// the command template. Names that clash with built-in helpers override them.
func WithFuncs(funcs template.FuncMap) Option {
	return func(c *config) {
		if c.funcs == nil {
			c.funcs = template.FuncMap{}
		}
		maps.Copy(c.funcs, funcs)
	}
}

// WithDryRun makes the tree generators render every template without creating
// the output directory or writing any file. Use it to validate templates
// against a real command tree, for instance in CI. It has no effect on
// [GenDocs], which writes to the caller's io.Writer.
func WithDryRun() Option {
	return func(c *config) { c.dryRun = true }
}

// WithHiddenFlags includes hidden flags in [CommandData.Flags] and
// [CommandData.InheritedFlags]. By default hidden flags are omitted, as in
// Cobra's help output and its own doc generators.
func WithHiddenFlags() Option {
	return func(c *config) { c.hiddenFlags = true }
}

// WithoutHelpFlag omits the --help flag that Cobra adds automatically to every
// command. Help flags defined explicitly by the application are still included.
func WithoutHelpFlag() Option {
	return func(c *config) { c.helpFlag = false }
}

// builtinFuncs returns the helper functions available to all templates.
// Functions that take a string argument take it last so they compose in
// pipelines, e.g. {{ .Short | trimSuffix "." }}.
func builtinFuncs() template.FuncMap {
	return template.FuncMap{
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
		"slug":       slug,
		"anchor":     anchor,
		"titleCase":  titleCase,
		"trimPrefix": func(prefix, s string) string { return strings.TrimPrefix(s, prefix) },
		"trimSuffix": func(suffix, s string) string { return strings.TrimSuffix(s, suffix) },
		"trimSpace":  strings.TrimSpace,
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"join":       func(sep string, elems []string) string { return strings.Join(elems, sep) },
		"replace":    func(old, new, s string) string { return strings.ReplaceAll(s, old, new) },
	}
}

// slug lowercases s, replaces runs of non-alphanumeric characters with a single
// dash, and trims dashes from both ends: "My App sub" -> "my-app-sub".
func slug(s string) string {
	return slugify(s, false)
}

// anchor is like slug but keeps underscores: "ref_my app" -> "ref_my-app".
func anchor(s string) string {
	return slugify(s, true)
}

func slugify(s string, keepUnderscore bool) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || (keepUnderscore && r == '_'):
			b.WriteRune(r)
			dash = false
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// titleCase upper-cases the first letter of every whitespace-separated word,
// leaving the remaining letters untouched: "list all items" -> "List All Items".
func titleCase(s string) string {
	prevSpace := true
	var b strings.Builder
	for _, r := range s {
		if prevSpace && unicode.IsLetter(r) {
			r = unicode.ToTitle(r)
		}
		prevSpace = unicode.IsSpace(r)
		b.WriteRune(r)
	}
	return b.String()
}

func (c *config) funcMap() template.FuncMap {
	funcs := builtinFuncs()
	maps.Copy(funcs, c.funcs)
	return funcs
}

func (c *config) parseTemplate(name, content string) (*template.Template, error) {
	tmpl, err := template.New(name).
		Option("missingkey=error").
		Funcs(c.funcMap()).
		Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parsing %s template: %w", name, err)
	}
	return tmpl, nil
}

// GenDocs generates docs for a single command and writes them to w using the
// given template, which receives a [CommandData] value.
func GenDocs(cmd *cobra.Command, w io.Writer, templateContent string, opts ...Option) error {
	cfg := newConfig(opts)
	tmpl, err := cfg.parseTemplate("command", templateContent)
	if err != nil {
		return err
	}
	return cfg.genDocs(cmd, w, tmpl)
}

func (c *config) genDocs(cmd *cobra.Command, w io.Writer, tmpl *template.Template) error {
	buf := new(bytes.Buffer)
	if err := executeTemplate(tmpl, buf, c.commandData(cmd)); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}

func refFor(cmd *cobra.Command) string {
	return "ref_" + strings.ReplaceAll(cmd.CommandPath(), " ", "_")
}

func (c *config) fileFor(cmd *cobra.Command) string {
	return strings.ReplaceAll(cmd.CommandPath(), " ", "-") + c.ext
}

func (c *config) summary(cmd *cobra.Command) CommandSummary {
	return CommandSummary{
		Name:  cmd.CommandPath(),
		Short: cmd.Short,
		Ref:   refFor(cmd),
		File:  c.fileFor(cmd),
	}
}

// documented reports whether a command gets its own documentation: it must be
// available (not hidden, deprecated, or the help command) and not a plain help topic.
func documented(cmd *cobra.Command) bool {
	return cmd.IsAvailableCommand() && !cmd.IsAdditionalHelpTopicCommand()
}

// commandData extracts the template data for cmd.
func (c *config) commandData(cmd *cobra.Command) CommandData {
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	name := cmd.CommandPath()
	long := cmd.Long
	if long == "" {
		long = cmd.Short
	}

	relatedCommands := []string{}
	if related, exists := cmd.Annotations["related"]; exists {
		relatedCommands = strings.Split(related, ",")
		for i, rc := range relatedCommands {
			relatedCommands[i] = strings.TrimSpace(rc)
		}
	}

	var subcommands []CommandSummary
	for _, sub := range cmd.Commands() {
		if documented(sub) {
			subcommands = append(subcommands, c.summary(sub))
		}
	}

	parent := ""
	if cmd.HasParent() {
		parent = cmd.Parent().CommandPath()
	}

	return CommandData{
		Ref:             refFor(cmd),
		CommandName:     name,
		File:            c.fileFor(cmd),
		Short:           cmd.Short,
		Long:            long,
		Synopsis:        cmd.UseLine(),
		Examples:        c.parser.Parse(cmd.Example),
		Flags:           c.flagInfos(cmd.NonInheritedFlags()),
		InheritedFlags:  c.flagInfos(cmd.InheritedFlags()),
		HeadingLen:      utf8.RuneCountInString(name),
		RelatedCommands: relatedCommands,
		Aliases:         cmd.Aliases,
		Deprecated:      cmd.Deprecated,
		Runnable:        cmd.Runnable(),
		Parent:          parent,
		Subcommands:     subcommands,
		Annotations:     cmd.Annotations,
	}
}

func (c *config) flagInfos(flags *pflag.FlagSet) []FlagInfo {
	var infos []FlagInfo
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden && !c.hiddenFlags {
			return
		}
		_, setByCobra := flag.Annotations[cobra.FlagSetByCobraAnnotation]
		if setByCobra && !c.helpFlag {
			return
		}
		_, required := flag.Annotations[cobra.BashCompOneRequiredFlag]
		infos = append(infos, FlagInfo{
			Name:                flag.Name,
			Shorthand:           flag.Shorthand,
			Usage:               flag.Usage,
			DefaultValue:        flag.DefValue,
			Type:                flag.Value.Type(),
			NoOptDefVal:         flag.NoOptDefVal,
			Deprecated:          flag.Deprecated,
			ShorthandDeprecated: flag.ShorthandDeprecated,
			Hidden:              flag.Hidden,
			Required:            required,
			SetByCobra:          setByCobra,
		})
	})
	return infos
}

// GenMarkdownTree generates Markdown documentation (.md files) for every
// subcommand of cmd into dir, plus an index file. See genDocsTree for details.
func GenMarkdownTree(
	cmd *cobra.Command,
	dir string,
	templates TemplateInfo,
	filePrepender func(string) string,
	opts ...Option,
) error {
	return genDocsTree(cmd, dir, templates, filePrepender, ".md", opts)
}

// GenRSTTree generates reStructuredText documentation (.rst files) for every
// subcommand of cmd into dir, plus an index file. See genDocsTree for details.
func GenRSTTree(
	cmd *cobra.Command,
	dir string,
	templates TemplateInfo,
	filePrepender func(string) string,
	opts ...Option,
) error {
	return genDocsTree(cmd, dir, templates, filePrepender, ".rst", opts)
}

// genDocsTree generates docs for a subcommand tree, skipping the root command
// itself and any hidden, deprecated, or help-topic commands. Each command is
// written to "<command path with dashes><ext>" in dir, preceded by the output
// of filePrepender (which receives the full file path and may be nil). The
// index template is rendered to templates.IndexFileName last.
func genDocsTree(
	cmd *cobra.Command,
	dir string,
	templates TemplateInfo,
	filePrepender func(string) string,
	ext string,
	opts []Option,
) error {
	if err := templates.Validate(); err != nil {
		return fmt.Errorf("invalid template configuration: %w", err)
	}
	cfg := newConfig(opts)
	cfg.ext = ext

	commandTmpl, err := cfg.parseTemplate("command", templates.SingleCommandTemplate)
	if err != nil {
		return err
	}
	indexTmpl, err := cfg.parseTemplate("index", templates.IndexTemplate)
	if err != nil {
		return err
	}
	if filePrepender == nil {
		filePrepender = func(string) string { return "" }
	}

	if !cfg.dryRun {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	var files []string
	var commands []CommandSummary

	var generateDocs func(*cobra.Command) error
	generateDocs = func(c *cobra.Command) error {
		if !documented(c) {
			return nil
		}

		basename := cfg.fileFor(c)
		filename := filepath.Join(dir, basename)

		buf := new(bytes.Buffer)
		buf.WriteString(filePrepender(filename))
		if err := cfg.genDocs(c, buf, commandTmpl); err != nil {
			return fmt.Errorf("generating %s: %w", basename, err)
		}
		if err := cfg.writeFile(filename, buf.Bytes()); err != nil {
			return err
		}

		files = append(files, basename)
		commands = append(commands, cfg.summary(c))

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

	data := IndexData{
		Root:     cmd.Name(),
		Files:    files,
		Commands: commands,
	}
	buf := new(bytes.Buffer)
	if err := executeTemplate(indexTmpl, buf, data); err != nil {
		return fmt.Errorf("generating %s: %w", templates.IndexFileName, err)
	}
	return cfg.writeFile(filepath.Join(dir, templates.IndexFileName), buf.Bytes())
}

// writeFile writes content to path unless the configuration is a dry run.
func (c *config) writeFile(path string, content []byte) error {
	if c.dryRun {
		return nil
	}
	// Generated documentation is meant to be read (and committed) by others,
	// so use the conventional 0644 rather than a private mode.
	if err := os.WriteFile(filepath.Clean(path), content, 0644); err != nil { // #nosec G306
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

// ValidateTemplates checks that the templates parse and render. Both templates
// are executed against synthetic data: the command template against a fully
// populated [CommandData] (every list non-empty, every string set) and against
// a minimal one (so that {{else}} branches run too), and the index template
// against an [IndexData] listing both. No files are written.
//
// Pass the same options you use for generation so that functions added with
// [WithFuncs] are known to the templates. Only template code that is actually
// executed is checked; conditionals that depend on other data (for example
// custom annotations) may still fail at generation time.
func ValidateTemplates(templates TemplateInfo, opts ...Option) error {
	if err := templates.Validate(); err != nil {
		return fmt.Errorf("invalid template configuration: %w", err)
	}
	cfg := newConfig(opts)

	commandTmpl, err := cfg.parseTemplate("command", templates.SingleCommandTemplate)
	if err != nil {
		return err
	}
	indexTmpl, err := cfg.parseTemplate("index", templates.IndexTemplate)
	if err != nil {
		return err
	}

	full, minimal := syntheticCommands()
	for _, data := range []CommandData{full, minimal} {
		if err := executeTemplate(commandTmpl, io.Discard, data); err != nil {
			return fmt.Errorf("rendering command template for %q: %w", data.CommandName, err)
		}
	}
	index := IndexData{
		Root:  "app",
		Files: []string{full.File, minimal.File},
		Commands: []CommandSummary{
			{Name: full.CommandName, Short: full.Short, Ref: full.Ref, File: full.File},
			{Name: minimal.CommandName, Short: minimal.Short, Ref: minimal.Ref, File: minimal.File},
		},
	}
	if err := executeTemplate(indexTmpl, io.Discard, index); err != nil {
		return fmt.Errorf("rendering index template: %w", err)
	}
	return nil
}

// syntheticCommands returns a fully populated and a minimal CommandData for
// template validation.
func syntheticCommands() (full, minimal CommandData) {
	full = CommandData{
		Ref:         "ref_app_group_cmd",
		CommandName: "app group cmd",
		File:        "app-group-cmd",
		Short:       "Do the thing",
		Long:        "Do the thing, at length.\n\nSecond paragraph.",
		Synopsis:    "app group cmd [NAME] [flags]",
		Examples: []ExampleInfo{
			{Info: "Do it:", Usage: "  $ app group cmd"},
			{Info: "", Usage: "  $ app group cmd --force"},
		},
		Flags: []FlagInfo{
			{Name: "force", Shorthand: "f", Usage: "Force it", DefaultValue: "false", Type: "bool", Required: true},
			{Name: "name", Usage: "Name to use", DefaultValue: "default", Type: "string", NoOptDefVal: "x", Deprecated: "use --title", ShorthandDeprecated: "use --name"},
		},
		InheritedFlags: []FlagInfo{
			{Name: "help", Shorthand: "h", Usage: "help for cmd", DefaultValue: "false", Type: "bool", SetByCobra: true},
		},
		HeadingLen:      len("app group cmd"),
		RelatedCommands: []string{"app group", "app other"},
		Aliases:         []string{"c", "command"},
		Deprecated:      "use \"app group other\" instead",
		Runnable:        true,
		Parent:          "app group",
		Subcommands: []CommandSummary{
			{Name: "app group cmd sub", Short: "A subcommand", Ref: "ref_app_group_cmd_sub", File: "app-group-cmd-sub"},
		},
		Annotations: map[string]string{"related": "app group, app other"},
	}
	minimal = CommandData{
		Ref:             "ref_app_leaf",
		CommandName:     "app leaf",
		File:            "app-leaf",
		Short:           "Leaf",
		Long:            "Leaf",
		Synopsis:        "app leaf",
		HeadingLen:      len("app leaf"),
		RelatedCommands: []string{},
		Parent:          "app",
	}
	return full, minimal
}

// executeTemplate safely executes a template with panic recovery.
// It recovers from panics during template execution and converts them to errors.
func executeTemplate(tmpl *template.Template, w io.Writer, data any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("template execution panic: %v", r)
		}
	}()
	err = tmpl.Execute(w, data)
	return err
}

// Parse extracts structured examples from a raw example string.
// See [ExampleParser] for the detection rules.
func (p *ExampleParser) Parse(example string) []ExampleInfo {
	if example == "" {
		return nil
	}
	example = strings.ReplaceAll(example, "\r\n", "\n")

	prefixes := p.CommandPrefixes
	if len(prefixes) == 0 {
		prefixes = DefaultCommandPrefixes
	}
	minIndent := p.MinIndent
	if minIndent <= 0 {
		minIndent = DefaultMinIndent
	}
	separator := p.BlockSeparator
	if separator == "" {
		separator = DefaultBlockSeparator
	}

	entries := strings.Split(example, separator)
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
			if p.isCommandLine(line, prefixes, minIndent) {
				// First command line found - split here
				infoLines = lines[:i]
				usageLines = lines[i:]
				foundCommand = true
				break
			}
		}

		// Fallback: if no command prefix/indent detected, treat as usage-only
		if !foundCommand {
			usageLines = lines
		}

		info := ""
		if len(infoLines) > 0 {
			info = strings.TrimSpace(strings.Join(infoLines, "\n"))
		}
		results = append(results, ExampleInfo{
			Info:  info,
			Usage: strings.Join(usageLines, "\n"),
		})
	}

	return results
}

// isCommandLine determines if a line looks like a command based on prefixes or indentation.
func (p *ExampleParser) isCommandLine(line string, prefixes []string, minIndent int) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}

	if p.DisableIndentDetection {
		return false
	}

	// Count leading indentation; a tab is enough on its own.
	indent := 0
	for _, r := range line {
		switch r {
		case ' ':
			indent++
		case '\t':
			indent += minIndent
		default:
			return indent >= minIndent
		}
	}
	return false
}
