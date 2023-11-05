package main

import (
	"os"

	"github.com/leep-frog/command/sourcerer"
	"github.com/leep-frog/gocli"
)

func main() {
	os.Exit(sourcerer.Source([]sourcerer.CLI{gocli.CLI()}))
}
