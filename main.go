package main

import (
	"os"

	"github.com/szabado/ink/cli"
)

func main() {
	if err := cli.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
