package inspect

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// A directory holding any of these is a monorepo workspace root, so
// services in any member package group under it.
var workspaceMarkers = []string{
	"turbo.json",
	"nx.json",
	"pnpm-workspace.yaml",
	"yarn.lock",
	"go.work",
}

// A directory holding any of these is a project root candidate when no
// .git is found above the process cwd.
var regularMarkers = []string{
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

// markerSet is the list of markers one scan runs with: the workspace
// detectors are fixed, the regular list can gain user-configured names.
type markerSet struct {
	workspace []string
	regular   []string
}

// DetectProject maps a process working directory to the project it
// belongs to. The outermost .git wins, grouping monorepo services under
// one root; otherwise the outermost workspace marker (turbo.json, nx.json,
// pnpm-workspace.yaml, etc.); otherwise the nearest directory holding a
// regular marker file. The home directory itself is never a project.
// Returns "" when nothing plausible is found.
func DetectProject(cwd string) string {
	return detectProject(cwd, markerSet{workspace: workspaceMarkers, regular: regularMarkers})
}

func detectProject(cwd string, markers markerSet) string {
	if cwd == "" {
		return ""
	}
	home, _ := os.UserHomeDir()
	var gitRoot, workspaceRoot, markerRoot string
	for dir := filepath.Clean(cwd); ; {
		if dir != home {
			// .git may be a file in worktrees/submodules, so stat either way
			if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
				gitRoot = dir // keep walking: outermost wins
			}
			if markers.hasWorkspace(dir) {
				workspaceRoot = dir // keep walking: outermost wins
			}
			if markerRoot == "" && markers.hasRegular(dir) {
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
	if workspaceRoot != "" {
		return workspaceRoot
	}
	return markerRoot
}

func (m markerSet) hasWorkspace(dir string) bool {
	for _, name := range m.workspace {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	// Cargo workspace: check for [workspace] in Cargo.toml
	return hasCargoWorkspace(dir)
}

func (m markerSet) hasRegular(dir string) bool {
	for _, name := range m.regular {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func hasCargoWorkspace(dir string) bool {
	cargoToml := filepath.Join(dir, "Cargo.toml")
	data, err := os.ReadFile(cargoToml)
	if err != nil {
		return false
	}
	// a whole [workspace] line, not a substring — [workspace.dependencies]
	// and friends are section members, not the workspace root
	for line := range bytes.Lines(data) {
		if strings.TrimSpace(string(line)) == "[workspace]" {
			return true
		}
	}
	return false
}
