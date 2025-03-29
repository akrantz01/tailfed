package main

import (
	"os"

	"github.com/akrantz01/tailfed/internal/cli"
)

func main() {
	cli.Execute(os.Exit, os.Args[1:])
}
