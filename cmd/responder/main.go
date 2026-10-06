package main

import (
	"os"

	"github.com/rwittrup/responder-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute(cli.NewRootCmd()))
}
