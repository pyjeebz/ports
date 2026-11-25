package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestScriptName(t *testing.T) {
	for cmd, want := range map[string]string{
		"npm run dev":          "dev",
		"npm run-script dev":   "dev",
		"npm start":            "start",
		"npm install":          "",
		"pnpm dev":             "dev",
		"pnpm run storybook":   "storybook",
		"yarn dev":             "dev",
		"bun run dev":          "dev",
		"vite":                 "",
		"npx vite":             "",
		"/usr/bin/npm run dev": "dev",
	} {
		if got := scriptName(strings.Fields(cmd)); got != want {
			t.Errorf("scriptName(%q) = %q, want %q", cmd, got, want)
		}
	}
}

func TestScriptTextReadsPackageJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"dev":"vite --host"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := scriptText([]string{"npm", "run", "dev"}, dir); got != "vite --host" {
		t.Errorf("scriptText = %q, want the script behind npm run dev", got)
	}
	if got := scriptText([]string{"npm", "run", "missing"}, dir); got != "npm run missing" {
		t.Errorf("a missing script = %q, want the command itself", got)
	}
}

func TestRunFramework(t *testing.T) {
	for script, want := range map[string]string{
		"vite":                      "Vite",
		"vite dev --host":           "Vite",
		"node_modules/.bin/vite":    "Vite",
		"svelte-kit dev":            "Vite",
		"astro dev":                 "Astro",
		"ng serve":                  "Angular",
		"next dev":                  "",
		"nuxt dev":                  "",
		"uvicorn main:app --reload": "",
		"vitest":                    "",
		"echo invite":               "",
	} {
		if got := runFramework(script); got != want {
			t.Errorf("runFramework(%q) = %q, want %q", script, got, want)
		}
	}
}

func TestWithPortFlags(t *testing.T) {
	cases := []struct {
		cmd, framework, want string
	}{
		{"vite", "Vite", "vite --port 4123 --strictPort"},
		{"npm run dev", "Vite", "npm run dev -- --port 4123 --strictPort"},
		{"npm run dev -- --host", "Vite", "npm run dev -- --host --port 4123 --strictPort"},
		{"pnpm dev", "Astro", "pnpm dev --port 4123"},
		{"ng serve", "Angular", "ng serve --port 4123"},
		{"next dev", "", "next dev"},
		{"vite --port 5000", "Vite", "vite --port 5000"},
	}
	for _, c := range cases {
		got := strings.Join(withPortFlags(strings.Fields(c.cmd), c.framework, 4123), " ")
		if got != c.want {
			t.Errorf("withPortFlags(%q, %s) = %q, want %q", c.cmd, c.framework, got, c.want)
		}
	}
}

func TestOwnPort(t *testing.T) {
	for cmd, want := range map[string]uint32{
		"vite --port 5000":   5000,
		"vite --port=5001":   5001,
		"next dev -p 3001":   3001,
		"next dev":           0,
		"vite --port banana": 0,
	} {
		got, _ := ownPort(strings.Fields(cmd))
		if got != want {
			t.Errorf("ownPort(%q) = %d, want %d", cmd, got, want)
		}
	}
}

func TestPortEnv(t *testing.T) {
	env := portEnv(4123)
	for _, want := range []string{"PORT=4123", "FLASK_RUN_PORT=4123", "UVICORN_PORT=4123"} {
		if !slices.Contains(env, want) {
			t.Errorf("portEnv lacks %s: %v", want, env)
		}
	}
}

func TestPickPortStable(t *testing.T) {
	a, err := pickPort("shop")
	if err != nil {
		t.Skip(err)
	}
	b, _ := pickPort("shop")
	if a != b || a < 4000 || a > 4999 {
		t.Errorf("pickPort(shop) = %d then %d, want one stable port in 4000-4999", a, b)
	}
}

func TestNameArg(t *testing.T) {
	for in, want := range map[string]string{
		"shop":                          "shop",
		"API.Shop":                      "api.shop",
		"shop.localhost":                "shop",
		"http://api.shop.localhost:80/": "api.shop",
		"https://shop.localhost/x":      "shop",
	} {
		if got := nameArg(in); got != want {
			t.Errorf("nameArg(%q) = %q, want %q", in, got, want)
		}
	}
}
