// genman renders man(1) pages for the ports command tree. It's a release
// tool, not part of the binary: goreleaser runs it before packing the
// archives, so every release carries pages that document the exact flags
// that build shipped.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pyjeebz/ports/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	dir := flag.String("dir", "man", "directory to write the pages into")
	version := flag.String("version", "dev", "release version stamped into the page headers")
	flag.Parse()

	root := cmd.Root()
	root.DisableAutoGenTag = true // the footer names the release, not the generator

	if err := os.MkdirAll(*dir, 0o755); err != nil {
		log.Fatal(err)
	}
	header := &doc.GenManHeader{
		Title:   "PORTS",
		Section: "1",
		Source:  fmt.Sprintf("ports %s", *version),
		Manual:  "ports manual",
	}
	if err := doc.GenManTree(root, header, *dir); err != nil {
		log.Fatal(err)
	}

	pages, err := filepath.Glob(filepath.Join(*dir, "*.1"))
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range pages {
		fmt.Println("wrote", p)
	}
}
