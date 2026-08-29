package main

import (
	"github.com/pyjeebz/ports/cmd"
)

// version is stamped by goreleaser at release time.
var version = "dev"

func main() {
	cmd.Execute(version)
}
