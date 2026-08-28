package inspect

import (
	"testing"
	"time"
)

func TestParsePortFilters(t *testing.T) {
	cases := []struct {
		input    string
		want     []PortFilter
		wantErr  bool
	}{
		{"3000", []PortFilter{{Min: 3000, Max: 3000}}, false},
		{"3000,4000", []PortFilter{{Min: 3000, Max: 3000}, {Min: 4000, Max: 4000}}, false},
		{"3000-4000", []PortFilter{{Min: 3000, Max: 4000}}, false},
		{":3000,:8000", []PortFilter{{Min: 3000, Max: 3000}, {Min: 8000, Max: 8000}}, false},
		{"3000-4000,5000", []PortFilter{{Min: 3000, Max: 4000}, {Min: 5000, Max: 5000}}, false},
		{"", nil, false},
		{" 3000 , 4000 ", []PortFilter{{Min: 3000, Max: 3000}, {Min: 4000, Max: 4000}}, false},
		{"abc", nil, true},
		{"3000-", nil, true},
		{"-4000", nil, true},
		{"4000-3000", nil, true},
	}
	for _, c := range cases {
		got, err := parsePortFilters(c.input)
		if (err != nil) != c.wantErr {
			t.Errorf("parsePortFilters(%q) error = %v, wantErr %v", c.input, err, c.wantErr)
			continue
		}
		if !c.wantErr && len(got) != len(c.want) {
			t.Errorf("parsePortFilters(%q) = %v, want %v", c.input, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("parsePortFilters(%q)[%d] = %v, want %v", c.input, i, got[i], c.want[i])
			}
		}
	}
}

func TestParseUptimeFilter(t *testing.T) {
	cases := []struct {
		input    string
		wantMin  *time.Duration
		wantMax  *time.Duration
		wantErr  bool
	}{
		{">1h", durationPtr(time.Hour), nil, false},
		{">=1h", durationPtr(time.Hour), nil, false},
		{"<30m", nil, durationPtr(30 * time.Minute), false},
		{"<=30m", nil, durationPtr(30 * time.Minute), false},
		{"1h", durationPtr(time.Hour), nil, false}, // defaults to >=
		{"90", durationPtr(90 * time.Second), nil, false}, // seconds
		{"", nil, nil, false},
		{"invalid", nil, nil, true},
	}
	for _, c := range cases {
		min, max, err := parseUptimeFilter(c.input)
		if (err != nil) != c.wantErr {
			t.Errorf("parseUptimeFilter(%q) error = %v, wantErr %v", c.input, err, c.wantErr)
			continue
		}
		if !c.wantErr {
			if (min == nil) != (c.wantMin == nil) {
				t.Errorf("parseUptimeFilter(%q) min = %v, want %v", c.input, min, c.wantMin)
			}
			if min != nil && c.wantMin != nil && *min != *c.wantMin {
				t.Errorf("parseUptimeFilter(%q) min = %v, want %v", c.input, *min, *c.wantMin)
			}
			if (max == nil) != (c.wantMax == nil) {
				t.Errorf("parseUptimeFilter(%q) max = %v, want %v", c.input, max, c.wantMax)
			}
			if max != nil && c.wantMax != nil && *max != *c.wantMax {
				t.Errorf("parseUptimeFilter(%q) max = %v, want %v", c.input, *max, *c.wantMax)
			}
		}
	}
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
}

func TestFilterMatches(t *testing.T) {
	now := time.Now()
	s := Service{
		Port:       3000,
		Framework:  "Next.js",
		Project:    "/home/user/shop",
		ProjectName: "shop",
		Started:    now.Add(-2 * time.Hour),
	}

	f := &Filter{
		Ports: []PortFilter{{Min: 3000, Max: 3000}},
	}
	if !f.Matches(s) {
		t.Error("port filter should match")
	}

	f = &Filter{
		Ports: []PortFilter{{Min: 4000, Max: 5000}},
	}
	if f.Matches(s) {
		t.Error("port filter should not match")
	}

	f = &Filter{
		Frameworks: []string{"next.js"},
	}
	if !f.Matches(s) {
		t.Error("framework filter should match (case-insensitive)")
	}

	f = &Filter{
		Frameworks: []string{"vite"},
	}
	if f.Matches(s) {
		t.Error("framework filter should not match")
	}

	f = &Filter{
		Projects: []string{"shop"},
	}
	if !f.Matches(s) {
		t.Error("project filter should match on project name")
	}

	f = &Filter{
		Projects: []string{"/home/user/shop"},
	}
	if !f.Matches(s) {
		t.Error("project filter should match on project path")
	}

	f = &Filter{
		Projects: []string{"admin"},
	}
	if f.Matches(s) {
		t.Error("project filter should not match")
	}

	f = &Filter{
		UptimeMin: durationPtr(1 * time.Hour),
	}
	if !f.Matches(s) {
		t.Error("uptime min filter should match")
	}

	f = &Filter{
		UptimeMin: durationPtr(3 * time.Hour),
	}
	if f.Matches(s) {
		t.Error("uptime min filter should not match")
	}

	f = &Filter{
		UptimeMax: durationPtr(3 * time.Hour),
	}
	if !f.Matches(s) {
		t.Error("uptime max filter should match")
	}

	f = &Filter{
		UptimeMax: durationPtr(1 * time.Hour),
	}
	if f.Matches(s) {
		t.Error("uptime max filter should not match")
	}

	// Unknown uptime should not match any uptime filter
	sUnknown := Service{Port: 3000, Started: time.Time{}}
	f = &Filter{UptimeMin: durationPtr(1 * time.Hour)}
	if f.Matches(sUnknown) {
		t.Error("unknown uptime should not match uptime filter")
	}
}

func TestApplyFilter(t *testing.T) {
	services := []Service{
		{Port: 3000, Framework: "Next.js", Project: "/home/user/shop", ProjectName: "shop", Started: time.Now().Add(-2 * time.Hour)},
		{Port: 4000, Framework: "Vite", Project: "/home/user/admin", ProjectName: "admin", Started: time.Now().Add(-30 * time.Minute)},
		{Port: 5000, Framework: "PostgreSQL", Project: "docker: db", ProjectName: "db", Started: time.Now().Add(-24 * time.Hour)},
	}

	f := &Filter{Ports: []PortFilter{{Min: 3000, Max: 4000}}}
	filtered := ApplyFilter(services, f)
	if len(filtered) != 2 {
		t.Errorf("ApplyFilter port range = %d, want 2", len(filtered))
	}

	f = &Filter{Frameworks: []string{"next.js"}}
	filtered = ApplyFilter(services, f)
	if len(filtered) != 1 || filtered[0].Framework != "Next.js" {
		t.Errorf("ApplyFilter framework = %v, want [Next.js]", filtered)
	}

	f = &Filter{Projects: []string{"shop"}}
	filtered = ApplyFilter(services, f)
	if len(filtered) != 1 || filtered[0].ProjectName != "shop" {
		t.Errorf("ApplyFilter project = %v, want [shop]", filtered)
	}

	f = &Filter{UptimeMin: durationPtr(1 * time.Hour)}
	filtered = ApplyFilter(services, f)
	if len(filtered) != 2 {
		t.Errorf("ApplyFilter uptime min = %d, want 2", len(filtered))
	}
}