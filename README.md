# Gencodo

Gencodo is a format-agnostic documentation generator for CLI applications built with [Cobra](https://github.com/spf13/cobra).
It extracts command metadata, examples, and flags, formatting them into arbitrarily templated files for integration with your documentation workflows.

## Features

- Automatically generates documentation for Cobra-based CLI commands.
- Supports custom, format-agnostic templates for command documentation and index pages (see [Go templates](https://pkg.go.dev/text/template)).
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
    "github.com/yourusername/gencodo"
)

gencodo.GenDocsTree(rootCmd, "docs", templates, filePrepender, linkHandler)
```

- `rootCmd`: Your Cobra root command.
- `docs`: Output directory for documentation files.
- `templates`: Custom templates for documentation formatting.
- `filePrepender`: Function to prepend headers to files.
- `linkHandler`: Function to handle internal links.

## Example

If your CLI has a command structure like:

```
mycli
├── command1
│   ├── subcommand1
│   └── subcommand2
└── command2
```

Running `GenDocsTree` with the examples will generate documentation files for each command and an index file.

## Contributing

Contributions are welcome! Please submit a PR or open an issue for discussions.

## License

LGPLv3
