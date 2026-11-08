package inspect

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func portOf(t *testing.T, addr string) uint32 {
	t.Helper()
	_, p, _ := net.SplitHostPort(addr)
	n, err := strconv.Atoi(p)
	if err != nil {
		t.Fatal(err)
	}
	return uint32(n)
}

// The scheme comes from asking: a plain server is http://, a TLS one is
// https:// — even though it would also answer plain HTTP with a 400.
func TestWebURLScheme(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(404) })
	plain := httptest.NewServer(ok)
	defer plain.Close()
	secure := httptest.NewTLSServer(ok)
	defer secure.Close()

	if url, found := WebURL(portOf(t, plain.Listener.Addr().String()), "127.0.0.1"); !found || !strings.HasPrefix(url, "http://localhost:") {
		t.Errorf("plain server: got %q, %v", url, found)
	}
	if url, found := WebURL(portOf(t, secure.Listener.Addr().String()), "127.0.0.1"); !found || !strings.HasPrefix(url, "https://localhost:") {
		t.Errorf("TLS server: got %q, %v", url, found)
	}
}

// A listener that isn't a web server is no URL.
func TestWebURLNotHTTP(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			_, _ = c.Write([]byte("+OK not http\r\n"))
			_ = c.Close()
		}
	}()
	if url, found := WebURL(portOf(t, l.Addr().String()), ""); found {
		t.Errorf("a non-HTTP listener got %q", url)
	}
}
