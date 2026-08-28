package inspect

// Config holds optional configuration for scanning.
type Config struct {
	CustomFrameworks []FrameworkRule
	CustomProjectMarkers []string
}

// FrameworkRule mirrors the internal frameworkRule but exported for config.
type FrameworkRule struct {
	Name string
	Exe  []string
	Bin  []string
	Cmd  []string
}

// DefaultConfig returns a config with built-in defaults.
func DefaultConfig() *Config {
	return &Config{}
}

// MergeFrameworks merges custom framework rules with built-ins.
// Custom rules are prepended so they take priority.
func MergeFrameworks(custom []FrameworkRule) []frameworkRule {
	rules := make([]frameworkRule, 0, len(custom)+len(frameworkRules))
	for _, c := range custom {
		rules = append(rules, frameworkRule{
			name: c.Name,
			exe:  c.Exe,
			bin:  c.Bin,
			cmd:  c.Cmd,
		})
	}
	rules = append(rules, frameworkRules...)
	return rules
}

// MergeProjectMarkers merges custom project markers with built-ins.
func MergeProjectMarkers(custom []string) []string {
	markers := make([]string, 0, len(custom)+len(projectMarkers))
	markers = append(markers, custom...)
	markers = append(markers, projectMarkers...)
	return markers
}