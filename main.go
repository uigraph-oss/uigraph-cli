package main

import (
	"context"

	"github.com/uigraph-oss/uigraph-cli/cmd"
)

func main() {
	// Root command executes the CLI. We pass a background context for now; subcommands may override.
	ctx := context.Background()
	cmd.Execute(ctx)
}
