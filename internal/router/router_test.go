package router

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/pyjeebz/ports/internal/routes"
)

// backend is a dev server the router can be pointed at: a project folder
// called shop with a package.json, and an HTTP server on loopback.
type backend struct {
	project string
	port    uint32
	addr    string
}

func newBackend(t *testing.T, network, addr string, h http.Handler) backend {
	t.Helper()
	l, err := net.Listen(network, addr)
	if err != nil {
		t.Skipf("can't listen on %s: %v", addr, err)
	}
	srv := &http.Server{Handler: h, ReadHeaderTimeout: time.Second}
	go func() { _ = srv.Serve(l) }()
	t.Cleanup(func() { _ = srv.Close() })
	project := filepath.Join(t.TempDir(), "shop")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte(`{"name":"shop"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	host, _, _ := net.SplitHostPort(l.Addr().String())
	return backend{project: project, port: uint32(l.Addr().(*net.TCPAddr).Port), addr: host}
}

// routerFor is a router whose table holds b, under the name shop.
func routerFor(t *testing.T, bs ...backend) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	scan := func() ([]inspect.Service, error) {
		mu.Lock()
		defer mu.Unlock()
		var out []inspect.Service
		for _, b := range bs {
			out = append(out, inspect.Service{
				Port: b.port, Pid: int32(os.Getpid()), Addr: b.addr, Process: "node",
				Framework: "Vite", URL: fmt.Sprintf("http://localhost:%d", b.port),
				Project: b.project, Cwd: b.project, ProjectName: "shop",
			})
		}
		return out, nil
	}
	table := routes.NewTable(scan, nil, nil)
	if _, err := table.Refresh(); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(New(table, 80))
	t.Cleanup(srv.Close)
	return srv
}

func get(t *testing.T, srv *httptest.Server, host, accept string) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest("GET", srv.URL+"/", nil)
	req.Host = host
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return resp, string(body)
}

func TestProxiesByName(t *testing.T) {
	b := newBackend(t, "tcp4", "127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "host=%s xfh=%s xfp=%s", r.Host, r.Header.Get("X-Forwarded-Host"), r.Header.Get("X-Forwarded-Proto"))
	}))
	srv := routerFor(t, b)
	resp, body := get(t, srv, "Shop.localhost.", "")
	if resp.StatusCode != 200 {
		t.Fatalf("status %d, body %q", resp.StatusCode, body)
	}
	if body != "host=Shop.localhost. xfh=Shop.localhost. xfp=http" {
		t.Errorf("the dev server saw %q, want the name as Host and X-Forwarded-Host", body)
	}
}

func TestOtherLoopback(t *testing.T) {
	// bound to ::1 only, as Node does for "localhost" — while the scan
	// reports a wildcard address
	b := newBackend(t, "tcp6", "[::1]:0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "v6")
	}))
	b.addr = "0.0.0.0"
	srv := routerFor(t, b)
	if resp, body := get(t, srv, "shop.localhost", ""); resp.StatusCode != 200 || body != "v6" {
		t.Errorf("got %d %q, want the ::1 server's answer", resp.StatusCode, body)
	}
}

func TestRefusesOtherHosts(t *testing.T) {
	srv := routerFor(t)
	for _, host := range []string{"evil.example", "localhost.evil.example", "shop.localhost.evil.example"} {
		if resp, _ := get(t, srv, host, ""); resp.StatusCode != http.StatusMisdirectedRequest {
			t.Errorf("Host %s: status %d, want 421", host, resp.StatusCode)
		}
	}
}

func TestUnknownName(t *testing.T) {
	srv := routerFor(t)
	resp, body := get(t, srv, "nope.localhost", "")
	if resp.StatusCode != 404 || !strings.HasPrefix(body, "Nothing is called nope yet") {
		t.Errorf("got %d %q", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("a client that didn't ask for HTML got %s", ct)
	}
	resp, body = get(t, srv, "nope.localhost", "text/html,*/*")
	if !strings.Contains(body, `http-equiv="refresh"`) || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		t.Errorf("a browser should get a page that reloads itself, got %q", body)
	}
}

func TestIndex(t *testing.T) {
	b := newBackend(t, "tcp4", "127.0.0.1:0", http.NotFoundHandler())
	srv := routerFor(t, b)
	resp, body := get(t, srv, "localhost", "")
	if resp.StatusCode != 200 || !strings.Contains(body, "http://shop.localhost\t") {
		t.Errorf("index: %d %q", resp.StatusCode, body)
	}
}

func TestDown(t *testing.T) {
	b := newBackend(t, "tcp4", "127.0.0.1:0", http.NotFoundHandler())
	// nothing listens on the route's port any more
	l, _ := net.Listen("tcp4", "127.0.0.1:0")
	b.port = uint32(l.Addr().(*net.TCPAddr).Port)
	_ = l.Close()
	srv := routerFor(t, b)
	resp, body := get(t, srv, "shop.localhost", "")
	if resp.StatusCode != http.StatusBadGateway || !strings.HasPrefix(body, "shop isn't running") {
		t.Errorf("got %d %q", resp.StatusCode, body)
	}
}

func TestStreams(t *testing.T) {
	release := make(chan struct{})
	b := newBackend(t, "tcp4", "127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: one\n\n")
		w.(http.Flusher).Flush()
		<-release // the response stays open, as an event stream does
	}))
	defer close(release)
	srv := routerFor(t, b)
	req, _ := http.NewRequest("GET", srv.URL+"/", nil)
	req.Host = "shop.localhost"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	got := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(resp.Body).ReadString('\n')
		got <- line
	}()
	select {
	case line := <-got:
		if line != "data: one\n" {
			t.Errorf("first event %q", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the first event never came through: the router held the stream back")
	}
}

func TestWebSocketUpgrade(t *testing.T) {
	b := newBackend(t, "tcp4", "127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "websocket" {
			http.Error(w, "want an upgrade", 400)
			return
		}
		conn, rw, err := http.NewResponseController(w).Hijack()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
		_ = rw.Flush()
		// echo whatever comes, as raw bytes past the handshake
		line, _ := rw.ReadString('\n')
		_, _ = rw.WriteString("echo " + line)
		_ = rw.Flush()
	}))
	srv := routerFor(t, b)
	conn, err := net.Dial("tcp", strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = io.WriteString(conn, "GET /hmr HTTP/1.1\r\nHost: shop.localhost\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n")
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status %d, want 101", resp.StatusCode)
	}
	_, _ = io.WriteString(conn, "ping\n")
	line, err := br.ReadString('\n')
	if err != nil || line != "echo ping\n" {
		t.Errorf("after the upgrade got %q, %v", line, err)
	}
}

func TestHostname(t *testing.T) {
	for in, want := range map[string]string{
		"shop.localhost":      "shop.localhost",
		"SHOP.localhost:8480": "shop.localhost",
		"shop.localhost.":     "shop.localhost",
		"[::1]:80":            "::1",
		"localhost":           "localhost",
	} {
		if got := hostname(in); got != want {
			t.Errorf("hostname(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestURL(t *testing.T) {
	if got := URL("shop", 80); got != "http://shop.localhost" {
		t.Errorf("on 80: %s", got)
	}
	if got := URL("api.shop", 8480); got != "http://api.shop.localhost:8480" {
		t.Errorf("on 8480: %s", got)
	}
}
