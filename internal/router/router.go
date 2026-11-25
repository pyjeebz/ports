// Package router answers http://<name>.localhost: it reads the Host a
// request was sent to, finds the dev server holding that name in the
// route table, and proxies to it — plain requests, streamed responses
// and WebSocket upgrades alike, so hot reload works through it. It
// listens on loopback only and serves only .localhost names it knows; a
// request for any other host is refused, which is what keeps a web page
// from reaching a dev server by rebinding its own domain to 127.0.0.1.
package router

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pyjeebz/ports/internal/routes"
)

// Router is the http.Handler in front of a route table.
type Router struct {
	table *routes.Table
	port  int // the port the router listens on, for the links it prints
	proxy *httputil.ReverseProxy
}

type routeKey struct{}

// New is a router over table, listening on port.
func New(table *routes.Table, port int) *Router {
	rt := &Router{table: table, port: port}
	dialer := &net.Dialer{Timeout: 2 * time.Second, KeepAlive: 30 * time.Second}
	rt.proxy = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			r := pr.In.Context().Value(routeKey{}).(routes.Route)
			pr.SetURL(&url.URL{Scheme: "http", Host: r.Targets()[0]})
			// the dev server sees the name it was asked for, so the
			// links, redirects and cookies it makes carry the name too
			pr.Out.Host = pr.In.Host
			pr.SetXForwarded()
		},
		Transport: &http.Transport{
			Proxy:                 nil, // a local port, never through a proxy
			DialContext:           loopbackDialer(dialer),
			MaxIdleConnsPerHost:   16,
			IdleConnTimeout:       30 * time.Second,
			ResponseHeaderTimeout: 0, // a dev server compiling its first page takes its time
		},
		ErrorHandler: rt.unreachable,
	}
	return rt
}

// loopbackDialer dials a loopback address, then the other loopback on the
// same port: a server bound to "localhost" holds 127.0.0.1 or ::1, and
// the scan can't always tell which.
func loopbackDialer(d *net.Dialer) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := d.DialContext(ctx, network, addr)
		if err == nil {
			return conn, nil
		}
		host, port, splitErr := net.SplitHostPort(addr)
		if splitErr != nil {
			return nil, err
		}
		other := map[string]string{"127.0.0.1": "::1", "::1": "127.0.0.1"}[host]
		if other == "" || ctx.Err() != nil {
			return nil, err
		}
		if conn, err2 := d.DialContext(ctx, network, net.JoinHostPort(other, port)); err2 == nil {
			return conn, nil
		}
		return nil, err
	}
}

// ServeHTTP routes one request by its Host.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := hostname(r.Host)
	switch {
	case host == "localhost" || host == "127.0.0.1" || host == "::1":
		rt.index(w, r)
		return
	case !strings.HasSuffix(host, ".localhost"):
		// a page that rebinds its own domain to 127.0.0.1 arrives with
		// that domain as its Host: never ours to serve
		misdirected(w, r, host)
		return
	}
	name := strings.TrimSuffix(host, ".localhost")
	route, ok := rt.table.Lookup(name)
	if !ok {
		if was, ok := rt.table.Gone(name); ok {
			rt.down(w, r, was)
			return
		}
		rt.unknown(w, r, name)
		return
	}
	rt.proxy.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), routeKey{}, route)))
}

// hostname is the Host header without its port, lowercased, with no
// trailing dot — SHOP.localhost. and shop.localhost:80 are one name.
func hostname(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimSuffix(strings.Trim(host, "[]"), ".")
	return strings.ToLower(host)
}

// URL is where name answers through a router on port: no port in the
// URL on 80, which is the point of preferring it.
func URL(name string, port int) string {
	if port == 80 || port == 0 {
		return "http://" + name + ".localhost"
	}
	return "http://" + name + ".localhost:" + strconv.Itoa(port)
}

// unreachable answers when the route's server didn't: it stopped, or is
// moving to another port. The table rescans so the next request finds
// it if it moved.
func (rt *Router) unreachable(w http.ResponseWriter, r *http.Request, err error) {
	route, _ := r.Context().Value(routeKey{}).(routes.Route)
	go func() { _, _ = rt.table.Refresh() }()
	if errors.Is(err, context.Canceled) {
		return // the client went away; nobody to answer
	}
	rt.down(w, r, route)
}
