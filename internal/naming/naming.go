// Package naming turns a running app into the name it's reached by:
// shop.localhost for a project's main app, api.shop.localhost beside
// it, fix-ui.shop.localhost for the same app in a linked worktree. A
// name comes from things that don't move when a server restarts — the
// project, the app's directory, the branch — never from its port.
package naming

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// App is one web server to name: the project it belongs to, where it
// runs from, what it is, and the port it's on now — the port only breaks
// ties, so a name survives the server moving.
type App struct {
	Project   string // project root, as the PROJECT column finds it
	Cwd       string // the process's working directory
	Framework string
	Port      uint32
}

// Names gives each app its name, in the order given. Apps of one project
// share its base name: the main app gets the base alone (shop), the rest
// a role in front of it (api.shop). A linked worktree puts its branch
// between the two (fix-ui.shop, api.fix-ui.shop). Two repositories with
// the same base are told apart by claims, which remember who had it
// first. An app with no project gets "".
func Names(apps []App, claims *Claims) []string {
	names := make([]string, len(apps))
	type member struct {
		i    int
		role string
		main bool // in the running for the base name alone
	}
	type group struct {
		repo, base, branch string
		members            []member
	}
	groups := map[string]*group{}
	var order []string
	for i, a := range apps {
		if a.Project == "" {
			continue
		}
		root, repo, branch := checkout(a.Project, a.Cwd)
		key := root
		g, ok := groups[key]
		if !ok {
			g = &group{repo: repo, base: baseName(root, repo), branch: branch}
			groups[key] = g
			order = append(order, key)
		}
		role := dirRole(root, a.Cwd)
		g.members = append(g.members, member{i: i, role: role, main: role == ""})
	}

	for _, key := range order {
		g := groups[key]
		base := g.base
		if claims != nil {
			base = claims.Claim(g.repo, base)
		}
		if base == "" {
			continue
		}

		// more than one app runs from the project's own directory: the
		// frontends keep their claim on the base, a backend becomes api,
		// Storybook storybook
		var pool []int
		for j, m := range g.members {
			if m.main {
				pool = append(pool, j)
			}
		}
		if len(pool) > 1 {
			fronts := slices.DeleteFunc(slices.Clone(pool), func(j int) bool {
				return !frontends[apps[g.members[j].i].Framework]
			})
			if len(fronts) > 0 {
				for _, j := range pool {
					if !slices.Contains(fronts, j) {
						g.members[j].role, g.members[j].main = frameworkRole(apps[g.members[j].i].Framework), false
					}
				}
				pool = fronts
			}
		}
		// the lowest port takes the base; any other main candidate is
		// named for what it is
		slices.SortFunc(pool, func(a, b int) int {
			return cmp.Compare(apps[g.members[a].i].Port, apps[g.members[b].i].Port)
		})
		for k, j := range pool {
			if k > 0 {
				g.members[j].role = frameworkRole(apps[g.members[j].i].Framework)
			}
		}

		// a name two apps would share goes to the lower port; the next
		// one counts up (vite-2.shop)
		slices.SortStableFunc(g.members, func(a, b member) int {
			return cmp.Compare(apps[a.i].Port, apps[b.i].Port)
		})
		taken := map[string]bool{}
		for _, m := range g.members {
			name := join(m.role, g.branch, base)
			for n := 2; taken[name]; n++ {
				role := m.role
				if role == "" {
					role = frameworkRole(apps[m.i].Framework)
				}
				name = join(fmt.Sprintf("%s-%d", role, n), g.branch, base)
			}
			taken[name] = true
			names[m.i] = name
		}
	}
	return names
}

func join(parts ...string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ".")
}

// frontends are the frameworks that serve a page someone opens: when a
// project runs one beside a backend from the same directory, the page
// gets the project's name.
var frontends = map[string]bool{
	"Next.js": true, "Vite": true, "Nuxt": true, "Remix": true,
	"SvelteKit": true, "SolidStart": true, "Astro": true,
	"React (CRA)": true, "Webpack": true, "Angular": true,
}

// frameworkRole is what an app is called beside the main one when its
// directory doesn't say: storybook, api for a backend, else the
// framework itself.
func frameworkRole(framework string) string {
	switch {
	case framework == "Storybook":
		return "storybook"
	case frontends[framework]:
		return Label(framework)
	case framework == "":
		return "app"
	default:
		return "api"
	}
}

// mainDirs are the directory names a monorepo gives its main app: the
// app in apps/web is the project, not web.project.
var mainDirs = map[string]bool{
	"web": true, "app": true, "www": true, "site": true,
	"frontend": true, "client": true, "ui": true,
}

// holders are monorepo directories that hold apps rather than being one:
// apps/api is the api, not apps.
var holders = map[string]bool{
	"apps": true, "packages": true, "services": true, "projects": true,
	"modules": true, "src": true,
}

// dirRole is the role an app's directory gives it within its checkout:
// "" for the root itself or a main-app directory, else the directory's
// name — apps/api runs as api.
func dirRole(root, cwd string) string {
	if cwd == "" {
		return ""
	}
	rel, err := filepath.Rel(root, cwd)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	for seg := range strings.SplitSeq(filepath.ToSlash(rel), "/") {
		if holders[strings.ToLower(seg)] {
			continue
		}
		if mainDirs[strings.ToLower(seg)] {
			return ""
		}
		return Label(seg)
	}
	return ""
}

// checkout finds the tree an app runs from. Inside a linked git worktree
// that's the worktree, its branch, and the repository it belongs to — so
// the worktree's apps share the main checkout's base name and add the
// branch. Anywhere else it's the project itself, with no branch.
func checkout(project, cwd string) (root, repo, branch string) {
	start := cwd
	if start == "" || !within(start, project) {
		start = project
	}
	for dir := filepath.Clean(start); ; dir = filepath.Dir(dir) {
		if gitdir := linkedWorktree(dir); gitdir != "" {
			return dir, mainRepo(gitdir, dir), worktreeBranch(gitdir, dir)
		}
		if dir == filepath.Clean(project) || filepath.Dir(dir) == dir {
			break
		}
	}
	return project, project, ""
}

func within(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// linkedWorktree is the git directory a linked worktree's .git file
// points at, or "" when dir isn't one. Submodules have .git files too;
// theirs point into modules/, not worktrees/.
func linkedWorktree(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, ".git"))
	if err != nil {
		return ""
	}
	line, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir:")
	if !ok {
		return ""
	}
	gitdir := strings.TrimSpace(line)
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(dir, gitdir)
	}
	if filepath.Base(filepath.Dir(gitdir)) != "worktrees" {
		return ""
	}
	return gitdir
}

// mainRepo is the main checkout a worktree's git directory belongs to:
// <repo>/.git/worktrees/<name> lives in <repo>.
func mainRepo(gitdir, fallback string) string {
	common := filepath.Dir(filepath.Dir(gitdir))
	if filepath.Base(common) == ".git" {
		return filepath.Dir(common)
	}
	return fallback
}

// worktreeBranch is the label a worktree adds: the last segment of its
// branch (fix/ui → ui), or the worktree's folder when HEAD is detached.
func worktreeBranch(gitdir, dir string) string {
	data, err := os.ReadFile(filepath.Join(gitdir, "HEAD"))
	if err == nil {
		if ref, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "ref: refs/heads/"); ok {
			if l := Label(ref[strings.LastIndex(ref, "/")+1:]); l != "" {
				return l
			}
		}
	}
	return Label(filepath.Base(dir))
}

// baseName is what a project is called: its package.json name without
// the scope (@acme/shop is shop), else the folder of the repository it
// belongs to. Workspace roots often carry a placeholder name — root,
// monorepo — which says nothing, so those fall through to the folder.
func baseName(root, repo string) string {
	var pkg struct {
		Name string `json:"name"`
	}
	if data, err := os.ReadFile(filepath.Join(root, "package.json")); err == nil && json.Unmarshal(data, &pkg) == nil {
		name := pkg.Name
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		if l := Label(name); l != "" && !placeholders[l] {
			return l
		}
	}
	return Label(filepath.Base(repo))
}

var placeholders = map[string]bool{
	"root": true, "monorepo": true, "workspace": true, "workspaces": true,
	"app": true, "project": true,
}

// Label makes s one DNS label: lowercase letters, digits and single
// hyphens, at most 63 characters, never starting or ending in a hyphen.
// Anything else becomes a hyphen; "" when nothing usable is left.
func Label(s string) string {
	var b strings.Builder
	hyphen := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			hyphen = false
			continue
		}
		if !hyphen && b.Len() > 0 {
			b.WriteByte('-')
			hyphen = true
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if len(out) > 63 {
		out = strings.TrimRight(out[:63], "-")
	}
	return out
}
