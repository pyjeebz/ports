package inspect

import (
	"os"
	"path/filepath"
	"testing"
)

// Custom project markers from config must actually be used — a custom
// marker file makes a directory a project root, and custom framework
// rules take priority over the built-in table.
func TestCustomProjectMarkers(t *testing.T) {
	root := filepath.Join(t.TempDir(), "myapp")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("run:\n\tgo run ."), 0o644); err != nil {
		t.Fatal(err)
	}
	// a Makefile alone is not a built-in marker: the default table must
	// find nothing, the configured one must find the directory
	if got := DetectProject(root); got != "" {
		t.Fatalf("DetectProject with no custom marker = %q, want empty", got)
	}

	env := (&Config{CustomProjectMarkers: []string{"Makefile"}}).resolve()
	if got := detectProject(root, env.markers); got != root {
		t.Fatalf("detectProject with custom marker \"Makefile\" = %q, want %q", got, root)
	}
}

func TestCustomFrameworksTakePriority(t *testing.T) {
	// "Vite" is a built-in rule; a custom rule renaming it must win
	env := (&Config{CustomFrameworks: []FrameworkRule{
		{Name: "My Vite", Bin: []string{"vite"}},
	}}).resolve()
	if got := detectFramework(env.rules, "node", "node node_modules/.bin/vite"); got != "My Vite" {
		t.Fatalf("custom framework = %q, want %q", got, "My Vite")
	}
	// built-ins survive below the custom layer
	if got := detectFramework(env.rules, "node", "node node_modules/.bin/next dev"); got != "Next.js" {
		t.Fatalf("built-in framework = %q, want %q", got, "Next.js")
	}
}

func TestValidate(t *testing.T) {
	if err := (&Config{CustomFrameworks: []FrameworkRule{{Name: "Vend", Cmd: []string{"vend"}}}}).Validate(); err != nil {
		t.Errorf("a named rule with a matcher is valid: %v", err)
	}
	if err := (&Config{CustomFrameworks: []FrameworkRule{{Name: " ", Exe: []string{"vend"}}}}).Validate(); err == nil {
		t.Error("a blank name must be rejected")
	}
	if err := (&Config{CustomFrameworks: []FrameworkRule{{Name: "サーバー", Exe: []string{"vend"}}}}).Validate(); err == nil {
		t.Error("a name with no a-z/0-9 passed, but nothing could ever match it by name")
	}
	if err := (&Config{CustomFrameworks: []FrameworkRule{{Name: "Vend"}}}).Validate(); err == nil {
		t.Error("a rule with no exe, bin, or cmd matches everything and must be rejected")
	}
}
