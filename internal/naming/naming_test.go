package naming

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLabel(t *testing.T) {
	for in, want := range map[string]string{
		"shop":           "shop",
		"My Shop":        "my-shop",
		"Next.js":        "next-js",
		"__weird__name_": "weird-name",
		"日本":             "",
		"-x-":            "x",
		"a--b":           "a-b",
	} {
		if got := Label(in); got != want {
			t.Errorf("Label(%q) = %q, want %q", in, got, want)
		}
	}
	long := Label(strings.Repeat("b", 80))
	if len(long) > 63 {
		t.Errorf("Label kept %d characters, want at most 63", len(long))
	}
}

func TestNamesSingleApp(t *testing.T) {
	root := filepath.Join(t.TempDir(), "shop")
	write(t, filepath.Join(root, "go.mod"), "module shop\n")
	got := Names([]App{{Project: root, Cwd: root, Framework: "Gin", Port: 8080}}, nil)
	if got[0] != "shop" {
		t.Errorf("a project's only app = %q, want shop", got[0])
	}
}

func TestNamesPackageJSON(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "folder")
	write(t, filepath.Join(root, "package.json"), `{"name":"@acme/storefront"}`)
	other := filepath.Join(dir, "mono")
	write(t, filepath.Join(other, "package.json"), `{"name":"root"}`)
	got := Names([]App{
		{Project: root, Cwd: root, Framework: "Next.js", Port: 3000},
		{Project: other, Cwd: other, Framework: "Vite", Port: 5173},
	}, nil)
	if got[0] != "storefront" {
		t.Errorf("scoped package name = %q, want storefront", got[0])
	}
	if got[1] != "mono" {
		t.Errorf("placeholder package name = %q, want the folder, mono", got[1])
	}
}

func TestNamesMonorepo(t *testing.T) {
	root := filepath.Join(t.TempDir(), "shop")
	write(t, filepath.Join(root, "package.json"), `{"name":"shop"}`)
	got := Names([]App{
		{Project: root, Cwd: filepath.Join(root, "apps", "web"), Framework: "Next.js", Port: 3000},
		{Project: root, Cwd: filepath.Join(root, "apps", "api"), Framework: "NestJS", Port: 4000},
		{Project: root, Cwd: filepath.Join(root, "apps", "docs", "src"), Framework: "Astro", Port: 4321},
	}, nil)
	want := []string{"shop", "api.shop", "docs.shop"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("app %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNamesSameDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "shop")
	write(t, filepath.Join(root, "go.mod"), "module shop\n")
	got := Names([]App{
		{Project: root, Cwd: root, Framework: "FastAPI", Port: 8000},
		{Project: root, Cwd: root, Framework: "Vite", Port: 5173},
		{Project: root, Cwd: root, Framework: "Storybook", Port: 6006},
	}, nil)
	want := []string{"api.shop", "shop", "storybook.shop"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("app %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNamesTwoOfAKind(t *testing.T) {
	root := filepath.Join(t.TempDir(), "shop")
	write(t, filepath.Join(root, "go.mod"), "module shop\n")
	got := Names([]App{
		{Project: root, Cwd: root, Framework: "Vite", Port: 5174},
		{Project: root, Cwd: root, Framework: "Vite", Port: 5173},
		{Project: root, Cwd: root, Framework: "Vite", Port: 5175},
	}, nil)
	want := []string{"vite.shop", "shop", "vite-2.shop"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("app %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNamesNoProject(t *testing.T) {
	if got := Names([]App{{Cwd: "/", Framework: "Vite", Port: 5173}}, nil); got[0] != "" {
		t.Errorf("an app with no project = %q, want no name", got[0])
	}
}

// a linked worktree: the main checkout in shop/, the worktree beside it
// with the branch fix/ui checked out
func worktree(t *testing.T, branchRef string) (main, wt string) {
	t.Helper()
	dir := t.TempDir()
	main = filepath.Join(dir, "shop")
	wt = filepath.Join(dir, "shop-wt")
	gitdir := filepath.Join(main, ".git", "worktrees", "shop-wt")
	write(t, filepath.Join(gitdir, "HEAD"), branchRef+"\n")
	write(t, filepath.Join(main, ".git", "HEAD"), "ref: refs/heads/main\n")
	write(t, filepath.Join(wt, ".git"), "gitdir: "+gitdir+"\n")
	return main, wt
}

func TestNamesWorktree(t *testing.T) {
	main, wt := worktree(t, "ref: refs/heads/fix/ui")
	got := Names([]App{
		{Project: main, Cwd: main, Framework: "Next.js", Port: 3000},
		{Project: wt, Cwd: wt, Framework: "Next.js", Port: 3001},
		{Project: wt, Cwd: filepath.Join(wt, "api"), Framework: "FastAPI", Port: 8001},
	}, LoadClaimsAt(t))
	want := []string{"shop", "ui.shop", "api.ui.shop"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("app %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNamesDetachedWorktree(t *testing.T) {
	_, wt := worktree(t, "0123456789abcdef0123456789abcdef01234567")
	got := Names([]App{{Project: wt, Cwd: wt, Framework: "Vite", Port: 5173}}, nil)
	if got[0] != "shop-wt.shop" {
		t.Errorf("detached worktree = %q, want its folder as the branch, shop-wt.shop", got[0])
	}
}

func TestNamesSubmoduleIsNotAWorktree(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	write(t, filepath.Join(root, ".git"), "gitdir: ../.git/modules/lib\n")
	got := Names([]App{{Project: root, Cwd: root, Framework: "Vite", Port: 5173}}, nil)
	if got[0] != "lib" {
		t.Errorf("submodule = %q, want lib with no branch", got[0])
	}
}

// LoadClaimsAt is LoadClaims against a fresh state directory.
func LoadClaimsAt(t *testing.T) *Claims {
	t.Helper()
	t.Setenv("PORTS_STATE_DIR", t.TempDir())
	return LoadClaims()
}

func TestClaimsSameBaseName(t *testing.T) {
	c := LoadClaimsAt(t)
	dir := t.TempDir()
	first := filepath.Join(dir, "work", "shop")
	second := filepath.Join(dir, "client", "shop")
	for _, d := range []string{first, second} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if got := c.Claim(first, "shop"); got != "shop" {
		t.Fatalf("first claim = %q, want shop", got)
	}
	if got := c.Claim(second, "shop"); got != "shop-client" {
		t.Fatalf("second claim = %q, want shop-client", got)
	}
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}

	// remembered: the order the next scan sees them in changes nothing
	again := LoadClaims()
	if got := again.Claim(second, "shop"); got != "shop-client" {
		t.Errorf("second after reload = %q, want shop-client", got)
	}
	if got := again.Claim(first, "shop"); got != "shop" {
		t.Errorf("first after reload = %q, want shop", got)
	}
}

func TestClaimsReleasedWhenFolderGone(t *testing.T) {
	c := LoadClaimsAt(t)
	dir := t.TempDir()
	gone := filepath.Join(dir, "old", "shop")
	moved := filepath.Join(dir, "new", "shop")
	if err := os.MkdirAll(moved, 0o755); err != nil {
		t.Fatal(err)
	}
	c.Repos[gone] = "shop"
	if got := c.Claim(moved, "shop"); got != "shop" {
		t.Errorf("claim after the holder's folder went = %q, want shop", got)
	}
	if _, ok := c.Repos[gone]; ok {
		t.Error("the gone folder still holds a claim")
	}
}

func TestClaimsCountUp(t *testing.T) {
	c := LoadClaimsAt(t)
	dir := t.TempDir()
	var repos []string
	for _, p := range []string{"a/x/shop", "b/x/shop", "c/x/shop"} {
		d := filepath.Join(dir, p)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		repos = append(repos, d)
	}
	want := []string{"shop", "shop-x", "shop-x-2"}
	for i, r := range repos {
		if got := c.Claim(r, "shop"); got != want[i] {
			t.Errorf("claim %d = %q, want %q", i, got, want[i])
		}
	}
}
