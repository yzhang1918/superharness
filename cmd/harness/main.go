package main

import (
	"os"

	"github.com/catu-ai/easyharness/internal/cli"
)

func main() {
	app := cli.New(os.Stdout, os.Stderr)
	os.Exit(app.Run(os.Args[1:]))
}
