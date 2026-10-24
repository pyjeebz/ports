package inspect

import (
	"fmt"
	"strings"
)

// Config holds optional configuration for scanning. Everything in it
// extends the built-in detection; nothing here is required.
type Config struct {
	CustomFrameworks     []FrameworkRule
	CustomProjectMarkers []string
}

// FrameworkRule mirrors the internal frameworkRule but exported for config.
type FrameworkRule struct {
	Name string
	Exe  []string
	Bin  []string
	Cmd  []string
}

// Validate rejects rules that would mislabel the table: a rule with no
// name renders as a blank, and one with no exe, bin, or cmd matches every
// process on the machine.
func (cfg *Config) Validate() error {
	for i, r := range cfg.CustomFrameworks {
		if strings.TrimSpace(r.Name) == "" {
			return fmt.Errorf("frameworks[%d]: needs a name", i)
		}
		if len(r.Exe) == 0 && len(r.Bin) == 0 && len(r.Cmd) == 0 {
			return fmt.Errorf("frameworks[%d] (%s): needs at least one of exe, bin, or cmd — with none it would match every process", i, r.Name)
		}
	}
	return nil
}

// DefaultConfig returns a config with built-in defaults.
func DefaultConfig() *Config {
	return &Config{}
}

// scanEnv holds the resolved tables one scan runs with: the framework
// rule list and the project marker set, built-ins plus user extensions.
type scanEnv struct {
	rules   []frameworkRule
	markers markerSet
}

// resolve combines built-ins with the configured extensions. Custom
// framework rules are prepended so they take priority over built-ins;
// custom project markers are prepended to the regular marker list.
func (cfg *Config) resolve() scanEnv {
	env := scanEnv{
		rules:   make([]frameworkRule, 0, len(cfg.CustomFrameworks)+len(frameworkRules)),
		markers: markerSet{workspace: workspaceMarkers, regular: make([]string, 0, len(cfg.CustomProjectMarkers)+len(regularMarkers))},
	}
	for _, r := range cfg.CustomFrameworks {
		env.rules = append(env.rules, frameworkRule{name: r.Name, exe: r.Exe, bin: r.Bin, cmd: r.Cmd})
	}
	env.rules = append(env.rules, frameworkRules...)
	env.markers.regular = append(env.markers.regular, cfg.CustomProjectMarkers...)
	env.markers.regular = append(env.markers.regular, regularMarkers...)
	return env
}
