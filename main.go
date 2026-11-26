package main

import (
	"runtime/debug"
	"strings"

	"github.com/pyjeebz/ports/cmd"
)

// version is stamped by goreleaser at release time.
var version = "dev"

func main() {
	cmd.Execute(resolveVersion())
}

// resolveVersion is the stamped version, or — for a binary built with
// `go install github.com/pyjeebz/ports@v0.2.0`, which nothing stamps —
// the module version Go recorded in it.
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return strings.TrimPrefix(v, "v")
		}
	}
	return version
}
