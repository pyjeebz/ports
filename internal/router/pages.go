package router

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/pyjeebz/ports/internal/routes"
)

// The router's own pages: the index at http://localhost, an unknown
// name, and a known name whose server isn't answering. A browser gets a
// page that reloads itself every two seconds, so opening shop.localhost
// before the server is up lands on the app the moment it is. Anything
// else — curl, a script, an agent — gets the same words as plain text.

type pageRoute struct {
	Name, URL, Port, Framework, Project string
}

type page struct {
	Title   string
	Heading string
	Lines   []string
	Routes  []pageRoute
	Reload  bool
}

func (rt *Router) pageRoutes() []pageRoute {
	var out []pageRoute
	for _, r := range rt.table.All() {
		out = append(out, pageRoute{
			Name: r.Name, URL: URL(r.Name, rt.port), Port: fmt.Sprintf(":%d", r.Port),
			Framework: r.Framework, Project: inspect.ShortPath(r.Project),
		})
	}
	return out
}

func (rt *Router) index(w http.ResponseWriter, r *http.Request) {
	p := page{Title: "ports", Heading: "Dev servers on this machine", Routes: rt.pageRoutes(), Reload: true}
	if len(p.Routes) == 0 {
		p.Lines = []string{
			"Nothing is running yet.",
			"Start a dev server the way you always do (npm run dev) and it shows up here, by name.",
		}
	}
	render(w, r, http.StatusOK, p)
}

func (rt *Router) unknown(w http.ResponseWriter, r *http.Request, name string) {
	p := page{
		Title:   name + ".localhost",
		Heading: "Nothing is called " + name + " yet",
		Lines: []string{
			"Start its dev server and this page opens it on its own.",
			"A name comes from the project: its package.json name or its folder.",
		},
		Routes: rt.pageRoutes(),
		Reload: true,
	}
	render(w, r, http.StatusNotFound, p)
}

func (rt *Router) down(w http.ResponseWriter, r *http.Request, route routes.Route) {
	where := fmt.Sprintf("It was %s on :%d", route.Framework, route.Port)
	if route.Project != "" {
		where += ", in " + inspect.ShortPath(route.Project)
	}
	p := page{
		Title:   route.Name + ".localhost",
		Heading: route.Name + " isn't running",
		Lines:   []string{where + ".", "Start it again and this page reloads into it."},
		Reload:  true,
	}
	render(w, r, http.StatusBadGateway, p)
}

func misdirected(w http.ResponseWriter, _ *http.Request, host string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusMisdirectedRequest)
	_, _ = fmt.Fprintf(w, "ports routes *.localhost names only; this request was for %q\n", host)
}

func render(w http.ResponseWriter, r *http.Request, status int, p page) {
	w.Header().Set("Cache-Control", "no-store")
	if !strings.Contains(r.Header.Get("Accept"), "text/html") {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(status)
		_, _ = fmt.Fprintln(w, p.Heading)
		for _, l := range p.Lines {
			_, _ = fmt.Fprintln(w, l)
		}
		if len(p.Routes) > 0 {
			_, _ = fmt.Fprintln(w)
			for _, rt := range p.Routes {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", rt.URL, rt.Port, rt.Framework, rt.Project)
			}
		}
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = pageTmpl.Execute(w, p)
}

var pageTmpl = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
{{if .Reload}}<meta http-equiv="refresh" content="2">{{end}}
<title>{{.Title}}</title>
<style>
:root { --paper: #f7f3ea; --ink: #26231f; --muted: #6f685d; --rule: #d9d1c1; --link: #1f4f8a; }
@media (prefers-color-scheme: dark) {
  :root { --paper: #1c1b19; --ink: #ece6d9; --muted: #a39b8c; --rule: #3a3731; --link: #8fb8ea; }
}
* { box-sizing: border-box; }
body { margin: 0; background: var(--paper); color: var(--ink);
  font: 15px/1.6 ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace; }
main { max-width: 720px; margin: 0 auto; padding: 12vh 16px 48px; }
.mark { color: var(--muted); font-size: 13px; letter-spacing: .04em; }
h1 { font-size: 22px; font-weight: 600; margin: 6px 0 14px; }
p { margin: 0 0 6px; color: var(--muted); }
table { width: 100%; border-collapse: collapse; margin-top: 28px; }
td { padding: 9px 12px 9px 0; border-top: 1px dashed var(--rule); vertical-align: top; }
td.port, td.fw { color: var(--muted); white-space: nowrap; }
td.proj { color: var(--muted); overflow-wrap: anywhere; }
a { color: var(--link); text-decoration: none; }
a:hover { text-decoration: underline; }
@media (max-width: 560px) { td.proj { display: none; } }
</style>
</head>
<body>
<main>
<div class="mark">ports</div>
<h1>{{.Heading}}</h1>
{{range .Lines}}<p>{{.}}</p>
{{end}}
{{if .Routes}}<table>
{{range .Routes}}<tr><td><a href="{{.URL}}">{{.Name}}.localhost</a></td><td class="port">{{.Port}}</td><td class="fw">{{.Framework}}</td><td class="proj">{{.Project}}</td></tr>
{{end}}</table>{{end}}
</main>
</body>
</html>
`))
