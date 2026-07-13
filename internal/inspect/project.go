package inspect

import (
	"os"
	"path/filepath"
)

// A directory holding any of these is a project root candidate when no
// .git is found above the process cwd.
var projectMarkers = []string{
	"package.json",
	"go.mod",
	"pyproject.toml",
	"requirements.txt",
	"Cargo.toml",
	"Gemfile",
	"composer.json",
	"pom.xml",
	"build.gradle",
	"build.gradle.kts",
	"mix.exs",
	"deno.json",
	"deno.jsonc",
}

// DetectProject maps a process working directory to the project it
// belongs to. The outermost .git wins, grouping monorepo services under
// one root; otherwise the nearest directory holding a marker file. The
// home directory itself is never a project. Returns "" when nothing
// plausible is found.
func DetectProject(cwd string) string {
	if cwd == "" {
		return ""
	}
	home, _ := os.UserHomeDir()
	var gitRoot, markerRoot string
	for dir := filepath.Clean(cwd); ; {
		if dir != home {
			// .git may be a file in worktrees/submodules, so stat either way
			if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
				gitRoot = dir // keep walking: outermost wins
			}
			if markerRoot == "" && hasMarker(dir) {
				markerRoot = dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if gitRoot != "" {
		return gitRoot
	}
	return markerRoot
}

func hasMarker(dir string) bool {
	for _, m := range projectMarkers {
		if _, err := os.Stat(filepath.Join(dir, m)); err == nil {
			return true
		}
	}
	return false
}
