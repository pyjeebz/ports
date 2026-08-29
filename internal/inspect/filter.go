package inspect

import (
	"strconv"
	"strings"
	"time"
)

// Filter holds the filtering criteria for services.
type Filter struct {
	Ports      []PortFilter
	Frameworks []string
	Projects   []string
	UptimeMin  *time.Duration
	UptimeMax  *time.Duration
}

// PortFilter represents a single port or port range filter.
type PortFilter struct {
	Min uint32
	Max uint32
}

// ParseFilter parses the filter flags into a Filter struct.
func ParseFilter(ports, frameworks, projects, uptime string) (*Filter, error) {
	f := &Filter{}

	if ports != "" {
		pf, err := parsePortFilters(ports)
		if err != nil {
			return nil, err
		}
		f.Ports = pf
	}

	if frameworks != "" {
		for _, fw := range strings.Split(frameworks, ",") {
			fw = strings.TrimSpace(fw)
			if fw != "" {
				f.Frameworks = append(f.Frameworks, strings.ToLower(fw))
			}
		}
	}

	if projects != "" {
		for _, p := range strings.Split(projects, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				f.Projects = append(f.Projects, p)
			}
		}
	}

	if uptime != "" {
		min, max, err := parseUptimeFilter(uptime)
		if err != nil {
			return nil, err
		}
		f.UptimeMin = min
		f.UptimeMax = max
	}

	return f, nil
}

func parsePortFilters(s string) ([]PortFilter, error) {
	var filters []PortFilter
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Handle :3000 format
		part = strings.TrimPrefix(part, ":")

		if strings.Contains(part, "-") {
			// Range: 3000-4000
			bounds := strings.Split(part, "-")
			if len(bounds) != 2 {
				return nil, ErrInvalidPortFilter
			}
			min, err := strconv.ParseUint(strings.TrimSpace(bounds[0]), 10, 16)
			if err != nil {
				return nil, ErrInvalidPortFilter
			}
			max, err := strconv.ParseUint(strings.TrimSpace(bounds[1]), 10, 16)
			if err != nil {
				return nil, ErrInvalidPortFilter
			}
			if min > max {
				return nil, ErrInvalidPortFilter
			}
			filters = append(filters, PortFilter{Min: uint32(min), Max: uint32(max)})
		} else {
			// Single port
			p, err := strconv.ParseUint(part, 10, 16)
			if err != nil {
				return nil, ErrInvalidPortFilter
			}
			filters = append(filters, PortFilter{Min: uint32(p), Max: uint32(p)})
		}
	}
	return filters, nil
}

func parseUptimeFilter(s string) (*time.Duration, *time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil, nil
	}

	var op string
	if strings.HasPrefix(s, ">") || strings.HasPrefix(s, ">=") || strings.HasPrefix(s, "<") || strings.HasPrefix(s, "<=") {
		if strings.HasPrefix(s, ">=") || strings.HasPrefix(s, "<=") {
			op = s[:2]
			s = strings.TrimSpace(s[2:])
		} else {
			op = s[:1]
			s = strings.TrimSpace(s[1:])
		}
	} else {
		// Exact match treated as >=
		op = ">="
	}

	d, err := parseDuration(s)
	if err != nil {
		return nil, nil, err
	}

	switch op {
	case ">":
		return &d, nil, nil
	case ">=":
		return &d, nil, nil
	case "<":
		return nil, &d, nil
	case "<=":
		return nil, &d, nil
	default:
		return &d, nil, nil
	}
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrInvalidUptimeFilter
	}

	// Try parsing as Go duration first (1h, 30m, etc.)
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Try parsing as number of seconds
	if sec, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Duration(sec) * time.Second, nil
	}

	return 0, ErrInvalidUptimeFilter
}

// Matches reports whether a service matches the filter.
func (f *Filter) Matches(s Service) bool {
	if f == nil {
		return true
	}

	// Port filter
	if len(f.Ports) > 0 {
		matched := false
		for _, pf := range f.Ports {
			if s.Port >= pf.Min && s.Port <= pf.Max {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Framework filter (exact match, case-insensitive)
	if len(f.Frameworks) > 0 {
		matched := false
		sfw := strings.ToLower(s.Framework)
		for _, fw := range f.Frameworks {
			if sfw == fw {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Project filter (substring match on project name or path)
	if len(f.Projects) > 0 {
		matched := false
		for _, proj := range f.Projects {
			projLower := strings.ToLower(proj)
			if strings.Contains(strings.ToLower(s.Project), projLower) ||
				strings.Contains(strings.ToLower(s.ProjectName), projLower) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Uptime filter
	if f.UptimeMin != nil || f.UptimeMax != nil {
		if s.Started.IsZero() {
			return false // unknown uptime doesn't match any filter
		}
		uptime := time.Since(s.Started)
		if f.UptimeMin != nil && uptime < *f.UptimeMin {
			return false
		}
		if f.UptimeMax != nil && uptime > *f.UptimeMax {
			return false
		}
	}

	return true
}

// ApplyFilter filters a slice of services.
func ApplyFilter(services []Service, f *Filter) []Service {
	if f == nil {
		return services
	}
	var out []Service
	for _, s := range services {
		if f.Matches(s) {
			out = append(out, s)
		}
	}
	return out
}

// Error types for filter parsing.
type filterError string

func (e filterError) Error() string { return string(e) }

var (
	ErrInvalidPortFilter   = filterError("invalid port filter")
	ErrInvalidUptimeFilter = filterError("invalid uptime filter")
)
