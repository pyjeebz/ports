package inspect

import "testing"

func TestMatchesName(t *testing.T) {
	s := Service{Process: "node", Framework: "Next.js", Project: "/home/x/shop", ProjectName: "shop"}
	for _, q := range []string{"shop", "SHOP", "next.js", "node"} {
		if !s.MatchesName(q) {
			t.Errorf("MatchesName(%q) = false, want true", q)
		}
	}
	// kill matching is exact: no partials, no path fragments
	for _, q := range []string{"sho", "next", "x/shop", ""} {
		if s.MatchesName(q) {
			t.Errorf("MatchesName(%q) = true, want false", q)
		}
	}
}

func TestMatchesQuery(t *testing.T) {
	s := Service{Process: "python3.12", Framework: "FastAPI", Project: "/home/x/shop", ProjectName: "shop"}
	for _, q := range []string{"fast", "PYTHON", "sho", "x/shop"} {
		if !s.MatchesQuery(q) {
			t.Errorf("MatchesQuery(%q) = false, want true", q)
		}
	}
	if s.MatchesQuery("postgres") {
		t.Error("MatchesQuery(postgres) = true, want false")
	}
}

// kill and find take the framework the way it's typed, like the filter.
func TestMatchesTypedFramework(t *testing.T) {
	s := Service{Port: 3000, Pid: 1, Process: "node", Framework: "Next.js"}
	if !s.MatchesName("nextjs") || !s.MatchesQuery("nextjs") {
		t.Error("nextjs should name and find Next.js")
	}
	if s.MatchesName("next") {
		t.Error("kill names exactly — next is not Next.js")
	}
}
