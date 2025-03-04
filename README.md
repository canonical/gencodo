# Gencodo

Gencodo is a format-agnostic documentation generator for CLI applications built with [Cobra](https://github.com/spf13/cobra).
It extracts command metadata, examples, and flags, formatting them into arbitrarily templated files for integration with your documentation workflows.

## Features

- Automatically generates documentation for Cobra-based CLI commands.
- Supports custom, format-agnostic templates for individual and index pages.
- Extracts structured examples and flag details from Cobra commands.
- Handles related commands and command hierarchy.


## Usage

To generate documentation for a CLI application:

```shell
go get github.com/canonical/gencodo@latest
```

In your code:

```go
import (
    "github.com/canonical/gencodo"
)

gencodo.GenDocsTree(rootCmd, "docs", templates, filePrepender, linkHandler)
```

- `rootCmd`: Your Cobra root command.
- `docs`: Output directory for documentation files.
- `filePrepender`: Function to prepend headers to files.
- `linkHandler`: Function to handle internal links.

These arguments are in line with [Cobra's own implementation](https://umarcor.github.io/cobra/#generating-restructured-text-docs-for-your-own-cobracommand);
the only addition is `templates`, the argument that sets up the names of the [custom templates](https://github.com/canonical/gencodo/blob/123f06acd914276b95254f829280ef5a83e25cff/gencodo.go#L29) used for documentation formatting.

One way to add them to your code is [embedding](https://pkg.go.dev/embed), for instance:

```go
//go:embed gendocs/cli.rst gendocs/command.rst
var templates embed.FS
```

For more usage examples, see the [test code](https://github.com/canonical/gencodo/blob/main/gencodo_test.go).


## Templates

The library uses [Go templates](https://pkg.go.dev/text/template), so anything they support is readily available;
there's no requirement to stick to a certain markup format (Markdown, reST, YAML, JSON, etc.).
Also, see the example templates under `examples/`.


## Contributing

Contributions are welcome! Please submit a PR or open an issue for discussions.

## License

[LGPLv3](https://www.gnu.org/licenses/lgpl-3.0.en.html#license-text)
