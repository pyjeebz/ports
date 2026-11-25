package router

import (
	"strings"
	"testing"

	"github.com/pyjeebz/ports/internal/inspect"
)

func TestDiagnoseQuietWithoutNames(t *testing.T) {
	t.Setenv("PORTS_STATE_DIR", t.TempDir())
	if f := Diagnose(nil); len(f) != 0 {
		t.Errorf("no router and nothing named: %+v, want nothing said", f)
	}
}

func TestDiagnoseNamesWithoutRouter(t *testing.T) {
	t.Setenv("PORTS_STATE_DIR", t.TempDir())
	f := Diagnose([]string{"shop", "api.shop"})
	if len(f) == 0 || f[0].Check != "router" || !strings.Contains(f[0].Fix, "ports proxy") {
		t.Fatalf("got %+v, want a router finding pointing at ports proxy", f)
	}
	if inspect.HasProblems(f) {
		t.Errorf("names without a router fail doctor: %+v", f)
	}
}
