package inspect

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func row(port uint32, pid int32, name string) Service {
	return Service{Port: port, Pid: pid, Process: name, Cwd: "/home/dev/shop", Project: "/home/dev/shop", ProjectName: "shop"}
}

func containerRow(port uint32, id, image string) Service {
	return Service{Port: port, Pid: 1, Process: "docker-proxy", Container: "shop-db", ContainerID: id, Runtime: "docker", Image: image, Addr: "0.0.0.0"}
}

// summarizeDiffs compresses diffs to "change:port" pairs — the ordering
// and the verbs are the contract; messages are spot-checked separately.
func summarizeDiffs(diffs []Diff) string {
	var out []string
	for _, d := range diffs {
		out = append(out, fmt.Sprintf("%s:%d", d.Change, d.Port))
	}
	return strings.Join(out, "|")
}

func TestDiffServices(t *testing.T) {
	cases := []struct {
		name string
		was  []Service
		now  []Service
		want string
	}{
		{
			name: "an unchanged machine is silent",
			was:  []Service{row(3000, 100, "node")},
			now:  []Service{row(3000, 100, "node")},
		},
		{
			name: "uptime drift is not a change",
			was:  []Service{{Port: 3000, Pid: 100, Process: "node", Started: time.Now().Add(-time.Minute)}},
			now:  []Service{{Port: 3000, Pid: 100, Process: "node", Started: time.Now().Add(-time.Hour)}},
		},
		{
			name: "new listener",
			now:  []Service{row(3000, 100, "node")},
			want: "new:3000",
		},
		{
			name: "gone listener",
			was:  []Service{row(3000, 100, "node")},
			want: "gone:3000",
		},
		{
			name: "restart keeps the port, changes the pid",
			was:  []Service{row(3000, 100, "node")},
			now:  []Service{row(3000, 200, "node")},
			want: "changed:3000",
		},
		{
			name: "another process took the port",
			was:  []Service{row(3000, 100, "node")},
			now:  []Service{row(3000, 200, "python")},
			want: "changed:3000",
		},
		{
			name: "container restart, same proxy pid story",
			was:  []Service{containerRow(5432, "abc123", "postgres:16")},
			now:  []Service{containerRow(5432, "def456", "postgres:16")},
			want: "changed:5432",
		},
		{
			name: "unchanged container, same id",
			was:  []Service{containerRow(5432, "abc123", "postgres:16")},
			now:  []Service{containerRow(5432, "abc123", "postgres:16")},
		},
		{
			name: "windows row keyed by the windows pid",
			was:  []Service{{Port: 3000, Process: "node", Windows: true, WindowsPid: 4242}},
			now:  []Service{{Port: 3000, Process: "node", Windows: true, WindowsPid: 4242}},
		},
		{
			name: "windows row restarted",
			was:  []Service{{Port: 3000, Process: "node", Windows: true, WindowsPid: 4242}},
			now:  []Service{{Port: 3000, Process: "node", Windows: true, WindowsPid: 5000}},
			want: "changed:3000",
		},
		{
			name: "unknowable rows compare by presence alone",
			was:  []Service{{Port: 3000, Pid: 0, Process: "?"}},
			now:  []Service{{Port: 3000, Pid: 0, Process: "?"}},
		},
		{
			name: "multiplicity change on a shared port",
			was:  []Service{row(3000, 100, "node")},
			now:  []Service{row(3000, 100, "node"), row(3000, 200, "node")},
			want: "changed:3000",
		},
		{
			name: "changes sort by port regardless of scan order",
			was:  []Service{row(9000, 100, "node"), row(3000, 100, "node")},
			now:  []Service{row(9000, 100, "node"), row(4000, 300, "python")},
			want: "gone:3000|new:4000",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := summarizeDiffs(DiffServices(c.was, c.now))
			if got != c.want {
				t.Errorf("DiffServices = %q, want %q", got, c.want)
			}
		})
	}
}

func TestDiffMessages(t *testing.T) {
	t.Run("restart names both pids", func(t *testing.T) {
		d := DiffServices([]Service{row(3000, 100, "node")}, []Service{row(3000, 200, "node")})
		if len(d) != 1 || d[0].Message != "restarted — pid 100 → 200" {
			t.Errorf("restart message = %v", d)
		}
	})
	t.Run("a takeover names both processes", func(t *testing.T) {
		d := DiffServices([]Service{row(3000, 100, "node")}, []Service{row(3000, 200, "python")})
		if len(d) != 1 || d[0].Message != "was node, now python" {
			t.Errorf("takeover message = %v", d)
		}
	})
	t.Run("container restart names both ids", func(t *testing.T) {
		d := DiffServices([]Service{containerRow(5432, "abc123", "postgres:16")}, []Service{containerRow(5432, "def456", "postgres:16")})
		if len(d) != 1 || d[0].Message != "restarted — container abc123 → def456" {
			t.Errorf("container restart message = %v", d)
		}
	})
	t.Run("multiplicity change says who survives", func(t *testing.T) {
		d := DiffServices([]Service{row(3000, 100, "node")}, []Service{row(3000, 100, "node"), row(3000, 200, "node")})
		if len(d) != 1 || d[0].Message != "now 2 processes (was 1) — pid 100 survives" {
			t.Errorf("multiplicity message = %v", d)
		}
	})
	t.Run("a gone row still names what left", func(t *testing.T) {
		d := DiffServices([]Service{row(5432, 1, "postgres")}, nil)
		if len(d) != 1 || d[0].Process != "postgres" || d[0].Project == "" {
			t.Errorf("gone row = %+v", d)
		}
	})
}

func TestParseSnapshot(t *testing.T) {
	at := time.Date(2026, 8, 31, 9, 14, 0, 0, time.UTC)

	t.Run("the envelope round-trips", func(t *testing.T) {
		data, err := json.Marshal(Snapshot{Version: snapshotVersion, GeneratedAt: at, Services: []Service{row(3000, 100, "node")}})
		if err != nil {
			t.Fatal(err)
		}
		snap, err := ParseSnapshot(data)
		if err != nil {
			t.Fatalf("ParseSnapshot: %v", err)
		}
		if !snap.GeneratedAt.Equal(at) || len(snap.Services) != 1 || snap.Services[0].Pid != 100 {
			t.Errorf("round trip = %+v", snap)
		}
	})

	t.Run("a plain json capture loads, with no moment", func(t *testing.T) {
		data, err := json.Marshal([]Service{row(3000, 100, "node")})
		if err != nil {
			t.Fatal(err)
		}
		snap, err := ParseSnapshot(data)
		if err != nil {
			t.Fatalf("ParseSnapshot of a bare array: %v", err)
		}
		if !snap.GeneratedAt.IsZero() || len(snap.Services) != 1 {
			t.Errorf("bare array = %+v", snap)
		}
	})

	t.Run("a newer version refuses rather than guess", func(t *testing.T) {
		data, err := json.Marshal(Snapshot{Version: snapshotVersion + 1, GeneratedAt: at})
		if err != nil {
			t.Fatal(err)
		}
		_, err = ParseSnapshot(data)
		if err == nil || !strings.Contains(err.Error(), "version") {
			t.Errorf("newer version error = %v", err)
		}
	})

	t.Run("an object that is not a snapshot names the version gap", func(t *testing.T) {
		_, err := ParseSnapshot([]byte(`{"answer": 42}`))
		if err == nil || !strings.Contains(err.Error(), "version") {
			t.Errorf("wrong-shape error = %v", err)
		}
	})

	t.Run("garbage says so", func(t *testing.T) {
		_, err := ParseSnapshot([]byte("not json"))
		if err == nil || !strings.Contains(err.Error(), "does not parse") {
			t.Errorf("garbage error = %v", err)
		}
	})

	t.Run("an empty file says so", func(t *testing.T) {
		_, err := ParseSnapshot([]byte{})
		if err == nil || !strings.Contains(err.Error(), "does not parse") {
			t.Errorf("empty error = %v", err)
		}
	})
}

func TestLoadSnapshotMissingFile(t *testing.T) {
	_, err := LoadSnapshot("/no/such/before.json")
	if err == nil || !strings.Contains(err.Error(), "cannot read snapshot") {
		t.Errorf("missing file error = %v", err)
	}
}