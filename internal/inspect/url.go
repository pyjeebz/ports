package inspect

import (
	"fmt"
	"net"
)

// webFrameworks are the dev servers whose every listening port speaks
// plain HTTP, so the table can hand over a URL without sending a request.
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
	"Rails": true, "Gin": true, "Echo": true, "Chi": true, "Fiber": true,
	"Phoenix": true, "Spring Boot": true, "Wrangler": true, "PocketBase": true,
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
		host = s.Addr
	}
	return fmt.Sprintf("http://%s", net.JoinHostPort(host, fmt.Sprint(s.Port)))
}

func fillURLs(services []Service) {
	for i := range services {
		services[i].URL = localURL(services[i])
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
