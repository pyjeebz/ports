package inspect

import (
	"path/filepath"
	"slices"
	"strings"
)

// A frameworkRule maps what a process looks like to the name a developer
// knows it by. First match wins, so specific frameworks come before the
// generic runtimes they run on (a Next.js process is a node process).
type frameworkRule struct {
	name string   // display name
	exe  []string // exe basename prefixes; empty = any exe
	bin  []string // exact basename (script extension stripped) of any cmdline token
	cmd  []string // raw cmdline substrings
}

var frameworkRules = []frameworkRule{
	// node frameworks
	{name: "Next.js", bin: []string{"next"}, cmd: []string{"next-server"}},
	{name: "Vite", bin: []string{"vite"}},
	{name: "Nuxt", bin: []string{"nuxt", "nuxi"}},
	{name: "Astro", bin: []string{"astro"}},
	{name: "React (CRA)", bin: []string{"react-scripts"}},
	{name: "Webpack", bin: []string{"webpack-dev-server"}, cmd: []string{"webpack serve"}},
	{name: "Storybook", bin: []string{"storybook", "start-storybook"}},
	// python servers
	{name: "FastAPI", bin: []string{"fastapi"}},
	{name: "Uvicorn", bin: []string{"uvicorn"}},
	{name: "Django", exe: []string{"python"}, cmd: []string{"manage.py", "django"}},
	{name: "Flask", bin: []string{"flask"}},
	{name: "Gunicorn", bin: []string{"gunicorn"}, cmd: []string{"gunicorn"}},
	// databases & servers — prefixes match both the host binary and the
	// docker image name (mysqld / mysql:8, redis-server / redis:7, …)
	{name: "PostgreSQL", exe: []string{"postgres"}},
	{name: "MySQL", exe: []string{"mysql"}},
	{name: "MariaDB", exe: []string{"mariadb"}},
	{name: "Redis", exe: []string{"redis"}},
	{name: "MongoDB", exe: []string{"mongo"}},
	{name: "nginx", exe: []string{"nginx"}},
	{name: "Apache", exe: []string{"httpd", "apache2"}},
	// generic runtimes last
	{name: "Go (go run)", cmd: []string{"/go-build"}},
	{name: "Node.js", exe: []string{"node"}},
}

// script extensions stripped before comparing a cmdline token against bin
// names, so node_modules/vite/bin/vite.js still reads as "vite".
var scriptExts = []string{".js", ".mjs", ".cjs", ".ts", ".py"}

// DetectFramework names the framework behind a process from its exe
// basename and cmdline. Returns "" when nothing in the table matches.
func DetectFramework(process, cmdline string) string {
	exe := strings.ToLower(process)
	cmd := strings.ToLower(cmdline)
	for _, r := range frameworkRules {
		if r.matches(exe, cmd) {
			return r.name
		}
	}
	return ""
}

func (r frameworkRule) matches(exe, cmd string) bool {
	if len(r.exe) > 0 && !hasPrefixIn(exe, r.exe) {
		return false
	}
	// bin and cmd are alternative positive signals: if either list is
	// set, at least one entry from their union must hit.
	if len(r.bin) == 0 && len(r.cmd) == 0 {
		return true
	}
	for _, frag := range r.cmd {
		if strings.Contains(cmd, frag) {
			return true
		}
	}
	if len(r.bin) > 0 && binMatches(cmd, r.bin) {
		return true
	}
	return false
}

func hasPrefixIn(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// binMatches reports whether any cmdline token names one of bins — exact
// basename comparison, not substring, so .bin/vitest never reads as vite.
func binMatches(cmd string, bins []string) bool {
	for tok := range strings.FieldsSeq(cmd) {
		base := filepath.Base(tok)
		for _, ext := range scriptExts {
			base = strings.TrimSuffix(base, ext)
		}
		if slices.Contains(bins, base) {
			return true
		}
	}
	return false
}
