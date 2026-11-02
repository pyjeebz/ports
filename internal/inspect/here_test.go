package inspect

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestWithin(t *testing.T) {
	root := filepath.FromSlash("/home/dev/shop")
	cases := map[string]bool{
		"/home/dev/shop":          true,
		"/home/dev/shop/api":      true,
		"/home/dev/shop/api/src":  true,
		"/home/dev/shop-admin":    false, // a sibling, not a member
		"/home/dev":               false,
		"/home/dev/other/shop":    false,
		"/home/dev/shop/../admin": false,
	}
	for path, want := range cases {
		if got := within(filepath.FromSlash(path), root); got != want {
			t.Errorf("within(%q, %q) = %v, want %v", path, root, got, want)
		}
	}
}

func TestHere(t *testing.T) {
	root := filepath.FromSlash("/home/dev/shop")
	rows := []Service{
		{Port: 3000, Pid: 1, Framework: "Next.js", Project: root, Cwd: filepath.Join(root, "web")},
		// a process whose own project detection stopped lower down still
		// belongs by its working directory
		{Port: 8000, Pid: 2, Framework: "FastAPI", Project: filepath.Join(root, "api"), Cwd: filepath.Join(root, "api")},
		{Port: 5432, Container: "shop-db", ComposeDir: root},
		{Port: 5173, Pid: 3, Framework: "Vite", Project: filepath.FromSlash("/home/dev/shop-admin")},
		{Port: 6379, Container: "cache"}, // no compose dir: belongs nowhere
		{Port: 4000, Windows: true, WindowsPid: 9, Process: "node"},
	}
	got := Here(rows, root)
	var ports []uint32
	for _, s := range got {
		ports = append(ports, s.Port)
	}
	want := []uint32{3000, 8000, 5432}
	if len(ports) != len(want) {
		t.Fatalf("Here kept %v, want %v", ports, want)
	}
	for i := range want {
		if ports[i] != want[i] {
			t.Fatalf("Here kept %v, want %v", ports, want)
		}
	}
}

func TestProjectRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, "shop")
	sub := filepath.Join(root, "api", "src")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ProjectRoot(nil, sub); got != root {
		t.Errorf("ProjectRoot(subdir) = %q, want %q", got, root)
	}
	bare := filepath.Join(home, "notes")
	if err := os.MkdirAll(bare, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ProjectRoot(nil, bare); got != bare {
		t.Errorf("a directory in no project is its own root: got %q", got)
	}
	// custom markers count, the same as in the PROJECT column
	if err := os.WriteFile(filepath.Join(bare, ".vendroot"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(bare, "deep")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ProjectRoot(&Config{CustomProjectMarkers: []string{".vendroot"}}, inner); got != bare {
		t.Errorf("custom marker root = %q, want %q", got, bare)
	}
}

func listenOn(t *testing.T, network, addr string) (net.Listener, uint32) {
	t.Helper()
	l, err := net.Listen(network, addr)
	if err != nil {
		t.Skipf("cannot listen on %s: %v", addr, err)
	}
	return l, uint32(l.Addr().(*net.TCPAddr).Port)
}

func TestListening(t *testing.T) {
	l, port := listenOn(t, "tcp4", "127.0.0.1:0")
	if !Listening(port) {
		t.Errorf("Listening(%d) = false with a listener open", port)
	}
	_ = l.Close()
	if Listening(port) {
		t.Errorf("Listening(%d) = true after the listener closed", port)
	}
}

// Node binds "localhost" to ::1 on many systems; a probe that only
// tries 127.0.0.1 would wait forever on such a dev server.
func TestServesHTTPOnEitherLoopback(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:0", "[::1]:0"} {
		l, port := listenOn(t, "tcp", addr)
		srv := &http.Server{Handler: http.NotFoundHandler()} // a 404 is proof enough
		go func() { _ = srv.Serve(l) }()
		if !ServesHTTP(port) {
			t.Errorf("ServesHTTP on %s = false", addr)
		}
		if !dialable(port) {
			t.Errorf("dialable on %s = false", addr)
		}
		_ = srv.Close()
	}

	// a listener that never speaks HTTP is not a web server
	l, port := listenOn(t, "tcp4", "127.0.0.1:0")
	defer func() { _ = l.Close() }()
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	if ServesHTTP(port) {
		t.Error("a listener that hangs up is not HTTP")
	}
}

// A root reached through a symlink (macOS's /tmp is /private/tmp) still
// finds the processes under it: the kernel reports their paths resolved,
// so `ports here || npm run dev` must not start a duplicate from /tmp/app.
func TestHereThroughSymlink(t *testing.T) {
	real := filepath.Join(t.TempDir(), "shop")
	if err := os.MkdirAll(filepath.Join(real, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "shop-link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("can't create a symlink here: %v", err)
	}
	resolved := RealPath(real) // t.TempDir may itself sit behind a link
	rows := []Service{
		{Port: 3000, Pid: 1, Project: resolved, Cwd: filepath.Join(resolved, "web")},
		{Port: 5432, Container: "shop-db", ComposeDir: link}, // compose recorded the link
		{Port: 9000, Pid: 2, Project: filepath.Join(t.TempDir(), "other")},
	}
	got := Here(rows, link)
	if len(got) != 2 || got[0].Port != 3000 || got[1].Port != 5432 {
		t.Fatalf("Here through a symlink = %v, want :3000 and :5432", got)
	}
	if RealPath(link) != resolved {
		t.Errorf("RealPath(%q) = %q, want %q", link, RealPath(link), resolved)
	}
	if missing := filepath.Join(link, "nope"); RealPath(missing) != filepath.Clean(missing) {
		t.Errorf("an unresolvable path should come back cleaned, got %q", RealPath(missing))
	}
}
