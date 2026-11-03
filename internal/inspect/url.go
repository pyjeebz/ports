package inspect

import (
	"fmt"
	"net"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// webFrameworks are the dev servers that serve their app over plain HTTP
// unless told otherwise, so the table can hand over a URL without sending
// a request — on the app's port only (fillURLs), never a debugger's.
// A list, not a probe: the listing stays side-effect free — no HEAD that
// makes a Next.js dev server compile a page, no stall on a database port
// that never answers — and fast under --watch. Doctor and ports open
// probe when asked about one port. Frameworks that speak other protocols
// too (nginx and Apache may be HTTPS, Mailpit also does SMTP, databases)
// are left out rather than guessed.
var webFrameworks = map[string]bool{
	"Remix": true, "SvelteKit": true, "SolidStart": true, "Astro": true,
	"Hono": true, "Elysia": true, "Next.js": true, "Vite": true, "Nuxt": true,
	"React (CRA)": true, "Webpack": true, "Storybook": true, "Angular": true,
	"NestJS": true, "Strapi": true, "Directus": true, "FastAPI": true,
	"Uvicorn": true, "Starlette": true, "Sanic": true, "Quart": true,
	"Django": true, "Flask": true, "Gunicorn": true, "Laravel": true,
	"Rails": true, "Gin": true, "Fiber": true,
	"Phoenix": true, "Spring Boot": true, "PocketBase": true,
	// left out: Wrangler runs several processes on random ports, and Echo
	// and Chi match on bare words common in command lines
}

// localURL is where a listener answers from this machine: localhost for
// a wildcard or loopback bind, the bound address otherwise. Empty when
// the row isn't a known web dev server, or sits across the WSL boundary.
func localURL(s Service) string {
	if s.Windows || !webFrameworks[s.Framework] {
		return ""
	}
	host := "localhost"
	switch s.Addr {
	case "", "0.0.0.0", "::", "127.0.0.1", "::1", "*":
	default:
		// a zone id (fe80::1%eth0) is %25 inside a URL
		host = strings.Replace(s.Addr, "%", "%25", 1)
	}
	return fmt.Sprintf("http://%s", net.JoinHostPort(host, fmt.Sprint(s.Port)))
}

// sidePorts are the non-HTTP ports a dev server's own process commonly
// opens beside the app: the Node inspector, Java's debug wire, debugpy,
// LiveReload. The framework label is the process's, so without this a
// `next dev --inspect` would show a URL on its debugger port.
var sidePorts = map[uint32]bool{9229: true, 9230: true, 5005: true, 5678: true, 35729: true}

var (
	portFlag = regexp.MustCompile(`(?:^|\s)(?:--port|-p)(?:=|\s+)?(\d{1,5})(?:\s|$)`)
	// HTTPS chosen on the command line: the app answers https://, and
	// the table never guesses a scheme it can't see
	tlsFlag = regexp.MustCompile(`(?:^|\s)--(?:ssl|ssl-keyfile|ssl-certfile|certfile|keyfile|cert|cert-file|https|experimental-https)(?:=|\s|$)`)
)

// fillURLs gives each web dev server one URL, on the port the app serves:
// the one its command line names with --port or -p (a process that names
// a port it doesn't hold gets none), else its lowest port that isn't a
// known side port. Every other port the process holds — a debugger, a
// second listener — stays without one.
func fillURLs(services []Service) {
	byPid := map[int32][]int{}
	for i, s := range services {
		if localURL(s) == "" || tlsFlag.MatchString(s.Cmdline) {
			continue
		}
		if !s.Known() { // no pid to group by: the row stands alone
			services[i].URL = localURL(s)
			continue
		}
		byPid[s.Pid] = append(byPid[s.Pid], i)
	}
	for _, rows := range byPid {
		pick := -1
		if m := portFlag.FindStringSubmatch(services[rows[0]].Cmdline); m != nil {
			// the command line names the app's port: that port or none.
			// `next dev -p 3000` is a parent holding only its inspector;
			// the app port is its child's, which gets the URL instead
			if p, err := strconv.ParseUint(m[1], 10, 16); err == nil {
				pick = slices.IndexFunc(rows, func(i int) bool { return services[i].Port == uint32(p) })
			}
			if pick >= 0 {
				services[rows[pick]].URL = localURL(services[rows[pick]])
			}
			continue
		}
		if pick < 0 {
			for j, i := range rows {
				if !sidePorts[services[i].Port] && (pick < 0 || services[i].Port < services[rows[pick]].Port) {
					pick = j
				}
			}
		}
		if pick >= 0 {
			services[rows[pick]].URL = localURL(services[rows[pick]])
		}
	}
}

// Hyperlinks turns table URLs into OSC 8 links a terminal makes
// clickable. The command layer sets it only for an interactive stdout
// without NO_COLOR; a pipe always gets the plain text.
var Hyperlinks bool

func hyperlink(url string) string {
	if !Hyperlinks || url == "" {
		return url
	}
	return "\x1b]8;;" + url + "\x1b\\" + url + "\x1b]8;;\x1b\\"
}
