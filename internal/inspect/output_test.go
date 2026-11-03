package inspect

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// captureStdout runs f with os.Stdout redirected and returns what f wrote.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	real, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		buf, _ := io.ReadAll(real) // read to EOF: one Read can race the writer
		done <- string(buf)
	}()
	f()
	os.Stdout = saved
	_ = w.Close()
	return <-done
}

// A nil scan is [] in JSON — the shape consumers script against, not null.
func TestWriteJSONEmptyIsArray(t *testing.T) {
	out := captureStdout(t, func() {
		if err := WriteJSON(nil); err != nil {
			t.Error(err)
		}
	})
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("WriteJSON(nil) = %q, want []", out)
	}
}

// Only a path *under* home gets the ~ — a sibling like /home/dev-tools
// would otherwise collapse into "~-tools" and rename the project.
func TestProjectHomeSiblingBoundary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home) // os.UserHomeDir reads this
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home) // ...but reads USERPROFILE here
	}
	sibling := home + "-tools"
	_ = os.MkdirAll(filepath.Join(sibling, "api"), 0o755)

	// the path form — doctor's header prints it; the table prints
	// names (TestTableProjectNames)
	out := ProjectLabel(Service{Port: 3000, Pid: 1, Process: "node", Project: filepath.Join(home, "shop")}) + "\n" +
		ProjectLabel(Service{Port: 4000, Pid: 2, Process: "node", Project: filepath.Join(sibling, "api")})
	if !strings.Contains(out, "~/shop") {
		t.Errorf("a project under home should shorten to ~: %q", out)
	}
	if short := "~-tools"; strings.Contains(out, short) {
		t.Errorf("a sibling of home must not shorten: %q contains %q", out, short)
	}
	if !strings.Contains(out, filepath.Join(sibling, "api")) {
		t.Errorf("a sibling of home should print in full: %q", out)
	}
}

func TestParseSortMode(t *testing.T) {
	for _, ok := range []string{"", "port", "uptime", "process", "project"} {
		if mode, err := ParseSortMode(ok); err != nil || mode != SortMode(ok) {
			t.Errorf("ParseSortMode(%q) = %q, %v; want it accepted", ok, mode, err)
		}
	}
	if _, err := ParseSortMode("name"); err == nil || !strings.Contains(err.Error(), "invalid sort") {
		t.Errorf("ParseSortMode(\"name\") = %v, want an invalid-sort error", err)
	}
}

// scan order is deliberately scrambled: every mode has to re-arrange it.
// The container and unattributed rows carry no ProjectName — the modes
// disagree on where those land, which is half of what's being tested.
var sortFixture = []Service{
	{Port: 5173, Framework: "Vite", ProjectName: "admin", Project: "/x/admin", Started: time.Now().Add(-3 * time.Hour)},
	{Port: 3000, Framework: "Next.js", ProjectName: "shop", Project: "/x/shop", Started: time.Now().Add(-14 * time.Minute)},
	{Port: 5432, Framework: "PostgreSQL", Container: "shop-db", Started: time.Now().Add(-48 * time.Hour)},
	{Port: 9229, Framework: "Node.js", Started: time.Time{}},
}

func TestSortRows(t *testing.T) {
	cases := []struct {
		sort SortMode
		want []int // ports, in display order
	}{
		// default: project rows first grouped by name, projectless last
		{SortDefault, []int{5173, 3000, 5432, 9229}},
		{SortPort, []int{3000, 5173, 5432, 9229}},
		// longest-lived first: 2d, 3h, 14m, unknown last
		{SortUptime, []int{5432, 5173, 3000, 9229}},
		// process names case-insensitively: next.js, node.js, postgresql, vite
		{SortProcess, []int{3000, 9229, 5432, 5173}},
		// plain lexicographic by project — no projectless-last rule
		{SortProject, []int{5432, 9229, 5173, 3000}},
	}
	for _, c := range cases {
		rows := sortRows(sortFixture, c.sort)
		got := make([]int, len(rows))
		for i, r := range rows {
			got[i] = int(r.Port)
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("sortRows(sort %q) = %v, want %v", c.sort, got, c.want)
		}
	}
}

// The table names a project directory by its name, not its path: a path
// outside home can run to dozens of characters and push the table off
// the screen. Two directories that share a name get just enough of their
// parents to tell apart; non-directory projects read as they always have.
func TestTableProjectNames(t *testing.T) {
	long := filepath.FromSlash("/tmp/claude-0/very/long/scratch/path/shop")
	work := filepath.FromSlash("/srv/work/api")
	oss := filepath.FromSlash("/srv/oss/api")
	out := captureStdout(t, func() {
		WriteTable([]Service{
			{Port: 3000, Pid: 1, Process: "node", Project: long, ProjectName: "shop"},
			{Port: 8000, Pid: 2, Process: "python", Project: long, ProjectName: "shop"},
			{Port: 4000, Pid: 3, Process: "node", Project: work, ProjectName: "api"},
			{Port: 4001, Pid: 4, Process: "node", Project: oss, ProjectName: "api"},
			{Port: 5432, Process: "docker-proxy", Container: "shop-db", ContainerID: "a", Runtime: "docker"},
			{Port: 3306, Process: "mysqld", User: "mysql", Unit: "mysql"},
			{Port: 9000, Pid: 5, Process: "x"},
		}, SortPort)
	})
	for _, want := range []string{"shop", "work/api", "oss/api", "docker: shop-db", "systemd: mysql", "—"} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "/tmp/claude-0") || strings.Contains(out, "/srv/") {
		t.Errorf("the table should name projects, not print their paths:\n%s", out)
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if len(line) > 100 {
			t.Errorf("line runs %d columns: %q", len(line), line)
		}
	}
}

func TestDistinctSuffixes(t *testing.T) {
	got := distinctSuffixes([]string{"/a/x/shop", "/b/x/shop", "/c/shop"})
	want := map[string]string{"/a/x/shop": "a/x/shop", "/b/x/shop": "b/x/shop", "/c/shop": "c/shop"}
	for p, w := range want {
		if got[p] != w {
			t.Errorf("distinctSuffixes[%q] = %q, want %q", p, got[p], w)
		}
	}
	if got := distinctSuffixes([]string{"/srv/shop"}); got["/srv/shop"] != "shop" {
		t.Errorf("a lone project reads as its name, got %q", got["/srv/shop"])
	}
	// a path that is a suffix of another all the way up reads in full
	got = distinctSuffixes([]string{"/a/shop", "/b/a/shop"})
	if got["/a/shop"] != "/a/shop" || got["/b/a/shop"] != "b/a/shop" {
		t.Errorf("nested suffix: %v", got)
	}
}

// A directory whose name is another row kind's label keeps its path, and
// one directory spelled two ways is one project.
func TestTableProjectNameEdges(t *testing.T) {
	win := filepath.FromSlash("/mnt/c/windows")
	shop := filepath.FromSlash("/srv/shop")
	out := captureStdout(t, func() {
		WriteTable([]Service{
			{Port: 3000, Pid: 1, Process: "node", Project: win},
			{Port: 4000, Process: "Ollama", Windows: true},
			{Port: 5000, Pid: 2, Process: "node", Project: shop},
			{Port: 5001, Pid: 3, Process: "node", Project: shop + string(filepath.Separator)},
		}, SortPort)
	})
	if !strings.Contains(out, ShortPath(win)) {
		t.Errorf("a project named windows should keep its path:\n%s", out)
	}
	if strings.Count(out, " shop ") != 2 || strings.Contains(out, ShortPath(shop)) {
		t.Errorf("one directory spelled two ways should read as one name:\n%s", out)
	}
}
