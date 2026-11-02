package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pyjeebz/ports/internal/inspect"
)

// withConfig runs loadConfig against a ports-config.yaml in a fresh
// working directory, with a HOME that has none, and restores the
// package config afterwards.
func withConfig(t *testing.T, yaml string) error {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Chdir(dir)
	if yaml != "" {
		if err := os.WriteFile(filepath.Join(dir, "ports-config.yaml"), []byte(yaml), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	saved := scanConfig
	scanConfig = inspect.Config{}
	t.Cleanup(func() { scanConfig = saved })
	return loadConfig()
}

func TestLoadConfig(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{name: "no file is the normal case"},
		{name: "a valid file", yaml: "frameworks:\n  - name: Vend\n    exe: [vend]\nprojects:\n  markers: [.vendroot]\n"},
		{name: "yaml that doesn't parse", yaml: "frameworks: [\n", wantErr: "ports-config"},
		{name: "a value of the wrong shape", yaml: "frameworks: vend\n", wantErr: "frameworks"},
		{name: "a rule with no name", yaml: "frameworks:\n  - exe: [vend]\n", wantErr: "needs a name"},
		{name: "a rule that matches everything", yaml: "frameworks:\n  - name: Vend\n", wantErr: "match every process"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := withConfig(t, c.yaml)
			switch {
			case c.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)):
				t.Fatalf("error = %v, want one mentioning %q", err, c.wantErr)
			}
		})
	}
}

func TestLoadConfigFeedsTheScan(t *testing.T) {
	if err := withConfig(t, "frameworks:\n  - name: Vend\n    exe: [vend]\nprojects:\n  markers: [.vendroot]\n"); err != nil {
		t.Fatal(err)
	}
	cfg := buildInspectConfig()
	if len(cfg.CustomFrameworks) != 1 || cfg.CustomFrameworks[0].Name != "Vend" {
		t.Errorf("frameworks = %+v", cfg.CustomFrameworks)
	}
	if len(cfg.CustomProjectMarkers) != 1 || cfg.CustomProjectMarkers[0] != ".vendroot" {
		t.Errorf("markers = %+v", cfg.CustomProjectMarkers)
	}
}
