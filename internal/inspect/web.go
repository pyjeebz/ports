package inspect

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// WebURL is the page a listener serves, found by asking it: one HEAD
// over HTTPS, then one over plain HTTP, on each address it answers from
// here (a bound address, or both loopbacks for a wildcard or loopback
// bind). Any answer counts — a 404 or a redirect is as much a web server
// as a 200. HTTPS is asked first because many HTTPS servers answer plain
// HTTP with a 400, which would read as http:// and hand the browser the
// wrong scheme. ok is false when nothing answers HTTP at all.
//
// Unlike the table's URL column, this sends a request: it's what
// `ports open` and doctor do when asked about one port, never a listing.
func WebURL(port uint32, addr string) (url string, ok bool) {
	hosts, shown := loopbacks, "localhost"
	switch addr {
	case "", "0.0.0.0", "::", "127.0.0.1", "::1", "*":
	default:
		hosts, shown = []string{addr}, strings.Replace(addr, "%", "%25", 1)
	}
	for _, scheme := range []string{"https", "http"} {
		for _, host := range hosts {
			if answers(scheme, net.JoinHostPort(host, fmt.Sprint(port))) {
				return fmt.Sprintf("%s://%s/", scheme, net.JoinHostPort(shown, fmt.Sprint(port))), true
			}
		}
	}
	return "", false
}

// webProbe is the client WebURL asks with: short, never following a
// redirect (the first answer is the proof), and trusting any certificate
// — a dev server's is self-signed, and nothing is sent but a HEAD.
var webProbe = &http.Client{
	Timeout: 500 * time.Millisecond,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // a HEAD to a local dev server
		Proxy:           nil,                                   // a local port, never through HTTPS_PROXY
	},
}

func answers(scheme, hostport string) bool {
	resp, err := webProbe.Head(scheme + "://" + hostport + "/")
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return true
}

// OnWSL reports whether ports runs inside WSL, where a browser is
// Windows' to open.
func OnWSL() bool { return onWSL() }
