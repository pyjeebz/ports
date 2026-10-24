package inspect

import (
	"path/filepath"
	"slices"
	"strings"
)

// A frameworkRule maps what a process looks like to the name a developer
// knows it by. First match wins, so specific frameworks come before the
// generic runtimes they run on (a Next.js process is a node process).
type frameworkRule struct {
	name string   // display name
	exe  []string // exe basenames, versions and extensions allowed (node, node.exe, python3); empty = any exe
	bin  []string // exact basename (script extension stripped) of any cmdline token
	cmd  []string // cmdline fragments: whole words, or substrings when they carry punctuation (see cmdHas)
}

var frameworkRules = []frameworkRule{
	// node frameworks
	{name: "Remix", bin: []string{"remix"}, cmd: []string{"@remix-run"}},
	{name: "SvelteKit", bin: []string{"svelte-kit", "sveltekit"}, cmd: []string{"@sveltejs/kit"}},
	{name: "SolidStart", bin: []string{"solid-start"}, cmd: []string{"@solidjs/start"}},
	{name: "Astro", bin: []string{"astro"}},
	{name: "Hono", bin: []string{"hono"}, cmd: []string{"hono"}},
	{name: "Elysia", bin: []string{"elysia"}, cmd: []string{"elysia"}},
	{name: "Next.js", bin: []string{"next"}, cmd: []string{"next-server"}},
	{name: "Vite", bin: []string{"vite"}},
	{name: "Nuxt", bin: []string{"nuxt", "nuxi"}},
	{name: "React (CRA)", bin: []string{"react-scripts"}},
	{name: "Webpack", bin: []string{"webpack-dev-server"}, cmd: []string{"webpack serve"}},
	{name: "Storybook", bin: []string{"storybook", "start-storybook"}},
	{name: "Expo", bin: []string{"expo", "expo-cli"}},
	{name: "React Native", bin: []string{"react-native", "metro"}},
	{name: "Angular", bin: []string{"ng"}, cmd: []string{"@angular"}},
	{name: "NestJS", bin: []string{"nest"}},
	{name: "Strapi", bin: []string{"strapi"}},
	{name: "Directus", bin: []string{"directus"}},
	{name: "Prisma", bin: []string{"prisma"}},
	{name: "Drizzle Kit", bin: []string{"drizzle-kit"}},
	// python servers
	{name: "FastAPI", bin: []string{"fastapi"}},
	{name: "Uvicorn", bin: []string{"uvicorn"}},
	{name: "Starlette", bin: []string{"starlette"}, cmd: []string{"starlette"}},
	{name: "Sanic", bin: []string{"sanic"}, cmd: []string{"sanic"}},
	{name: "Quart", bin: []string{"quart"}, cmd: []string{"quart"}},
	{name: "Django", exe: []string{"python"}, cmd: []string{"manage.py", "django"}},
	{name: "Flask", bin: []string{"flask"}},
	{name: "Gunicorn", bin: []string{"gunicorn"}, cmd: []string{"gunicorn"}},
	// php & ruby servers
	{name: "Laravel", exe: []string{"php"}, cmd: []string{"artisan"}},
	{name: "Rails", bin: []string{"puma", "rails", "rackup"}},
	// go frameworks
	{name: "Gin", bin: []string{"gin"}, cmd: []string{"github.com/gin-gonic/gin"}},
	{name: "Echo", bin: []string{"echo"}, cmd: []string{"github.com/labstack/echo"}},
	{name: "Chi", bin: []string{"chi"}, cmd: []string{"github.com/go-chi/chi"}},
	{name: "Fiber", bin: []string{"fiber"}, cmd: []string{"github.com/gofiber/fiber"}},
	{name: "PocketBase", bin: []string{"pocketbase"}, cmd: []string{"pocketbase"}},
	// databases & servers — each rule names the engine's real binaries,
	// versions allowed, so the same list reads both the host binary and
	// the docker image name (mysqld, mysql:8; redis-server, redis:7)
	{name: "SurrealDB", exe: []string{"surreal"}},
	{name: "ClickHouse", exe: []string{"clickhouse", "clickhouse-server"}},
	{name: "Dragonfly", exe: []string{"dragonfly"}},
	{name: "PostgreSQL", exe: []string{"postgres"}},
	{name: "MySQL", exe: []string{"mysql", "mysqld"}},
	{name: "MariaDB", exe: []string{"mariadb", "mariadbd"}},
	{name: "Redis", exe: []string{"redis", "redis-server", "redis-sentinel"}},
	{name: "MongoDB", exe: []string{"mongo", "mongod", "mongos"}},
	{name: "nginx", exe: []string{"nginx"}},
	{name: "Apache", exe: []string{"httpd", "apache2"}},
	{name: "Valkey", exe: []string{"valkey", "valkey-server"}},
	{name: "Memcached", exe: []string{"memcached"}},
	{name: "etcd", exe: []string{"etcd"}},
	{name: "NATS", exe: []string{"nats", "nats-server"}},
	{name: "MinIO", exe: []string{"minio"}},
	{name: "Consul", exe: []string{"consul"}},
	// erlang & the jvm — before their generic runtimes
	{name: "Phoenix", exe: []string{"beam", "beam.smp", "erl"}, cmd: []string{"phx.server"}},
	{name: "RabbitMQ", exe: []string{"beam", "beam.smp", "rabbitmq"}, cmd: []string{"rabbit"}},
	{name: "Spring Boot", exe: []string{"java"}, cmd: []string{"spring-boot", "springframework.boot"}},
	{name: ".NET", exe: []string{"dotnet"}},
	// local kubernetes & dev conveniences
	{name: "kube-apiserver", exe: []string{"kube-apiserver"}},
	{name: "kind", exe: []string{"kind"}},
	{name: "minikube", exe: []string{"minikube"}},
	{name: "Mailpit", exe: []string{"mailpit"}},
	{name: "Wrangler", bin: []string{"wrangler"}},
	{name: "Wrangler", exe: []string{"workerd"}},
	{name: "Supabase", bin: []string{"supabase"}},
	// rust frameworks
	{name: "Axum", bin: []string{"axum"}, cmd: []string{"axum"}},
	{name: "Actix-web", bin: []string{"actix-web"}, cmd: []string{"actix-web"}},
	// generic runtimes last
	{name: "Java", exe: []string{"java", "javaw"}},
	{name: "Bun", exe: []string{"bun"}},
	{name: "Deno", exe: []string{"deno"}},
	{name: "Go (go run)", cmd: []string{"/go-build"}},
	{name: "Node.js", exe: []string{"node", "nodejs"}},
}

// script extensions stripped before comparing a cmdline token against bin
// names, so node_modules/vite/bin/vite.js still reads as "vite".
var scriptExts = []string{".js", ".mjs", ".cjs", ".ts", ".py"}

// FrameworkKey is a framework name as people type it: case, spaces, and
// punctuation fall away, so nextjs names Next.js, springboot names
// Spring Boot, and react-cra names React (CRA).
func FrameworkKey(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if ('a' <= r && r <= 'z') || ('0' <= r && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// DetectFramework names the framework behind a process from its exe
// basename and cmdline. Returns "" when nothing in the table matches.
func DetectFramework(process, cmdline string) string {
	return detectFramework(frameworkRules, process, cmdline)
}

func detectFramework(rules []frameworkRule, process, cmdline string) string {
	exe := strings.ToLower(process)
	cmd := strings.ToLower(cmdline)
	for _, r := range rules {
		if r.matches(exe, cmd) {
			return r.name
		}
	}
	return ""
}

func (r frameworkRule) matches(exe, cmd string) bool {
	if len(r.exe) > 0 && !matchesBase(exe, r.exe) {
		return false
	}
	// bin and cmd are alternative positive signals: if either list is
	// set, at least one entry from their union must hit.
	if len(r.bin) == 0 && len(r.cmd) == 0 {
		return true
	}
	for _, frag := range r.cmd {
		if cmdHas(cmd, frag) {
			return true
		}
	}
	if len(r.bin) > 0 && binMatches(cmd, r.bin) {
		return true
	}
	return false
}

// cmdHas reports whether a cmdline carries frag. A fragment with path or
// package punctuation (@remix-run, github.com/gin-gonic/gin, manage.py)
// is specific enough as a substring; a bare word (hono, quart) must
// stand as a whole word, so ~/my-hono-app reads as Hono and
// ~/phonograph or ~/quarterly don't.
func cmdHas(cmd, frag string) bool {
	if !isWord(frag) {
		return strings.Contains(cmd, frag)
	}
	for i := 0; ; {
		j := strings.Index(cmd[i:], frag)
		if j < 0 {
			return false
		}
		start, end := i+j, i+j+len(frag)
		if (start == 0 || !isAlnum(cmd[start-1])) && (end == len(cmd) || !isAlnum(cmd[end])) {
			return true
		}
		i = start + 1
	}
}

func isWord(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; !isAlnum(c) && c != '-' && c != '_' {
			return false
		}
	}
	return s != ""
}

func isAlnum(c byte) bool {
	return ('a' <= c && c <= 'z') || ('0' <= c && c <= '9') || ('A' <= c && c <= 'Z')
}

// matchesBase reports whether s names one of the bases — the name itself,
// or a versioned or extension-bearing build of it (node.exe, python3,
// beam.smp). Hyphens and underscores end the match: node_exporter and
// node-exporter are different programs, not builds of node, and must never
// read as — or be stopped as — Node.js.
func matchesBase(s string, bases []string) bool {
	for _, b := range bases {
		if !strings.HasPrefix(s, b) {
			continue
		}
		rest := s[len(b):]
		if rest == "" {
			return true
		}
		if c := rest[0]; c == '.' || (c >= '0' && c <= '9') {
			return true
		}
	}
	return false
}

// binMatches reports whether any cmdline token names one of bins — exact
// basename comparison, not substring, so .bin/vitest never reads as vite.
func binMatches(cmd string, bins []string) bool {
	for tok := range strings.FieldsSeq(cmd) {
		base := filepath.Base(tok)
		for _, ext := range scriptExts {
			base = strings.TrimSuffix(base, ext)
		}
		if slices.Contains(bins, base) {
			return true
		}
	}
	return false
}
