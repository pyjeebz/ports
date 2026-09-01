package integration

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestMain(m *testing.M) {
	// Build the binary first. Without PORTS_BINARY, build to a temp path
	// — never into the repo — and always rebuild, so a stale binary can't
	// mask regressions.
	binary := os.Getenv("PORTS_BINARY")
	if binary == "" {
		binary = filepath.Join(os.TempDir(), "ports-integration")
		cmd := exec.Command("go", "build", "-o", binary, "github.com/pyjeebz/ports")
		if out, err := cmd.CombinedOutput(); err != nil {
			panic(fmt.Sprintf("failed to build ports: %v\n%s", err, out))
		}
	}
	os.Exit(m.Run())
}

func TestListPorts(t *testing.T) {
	port, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}

	fixture := HTTPFixture(t, port)
	defer fixture.Stop()
	fixture.WaitReady(t)

	stdout, stderr, code := RunPorts(t, "--json")
	if code != 0 {
		t.Fatalf("ports failed: %s", stderr)
	}

	var services []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &services); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	found := false
	for _, s := range services {
		if portNum, ok := s["port"].(float64); ok && int(portNum) == port {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find port %d in output", port)
	}
}

func TestFindCommand(t *testing.T) {
	port, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}

	fixture := NodeFixture(t, port, "test-server")
	defer fixture.Stop()
	fixture.WaitReady(t)

	// find uses substring matching on framework, process, project, etc.
	stdout, stderr, code := RunPorts(t, "find", "node", "--json")
	if code != 0 {
		t.Fatalf("ports find failed: %s", stderr)
	}

	var services []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &services); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(services) == 0 {
		t.Errorf("expected to find node process, got empty results")
	}
}

func TestFreeCommand(t *testing.T) {
	port, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}

	fixture := HTTPFixture(t, port)
	defer fixture.Stop()
	fixture.WaitReady(t)

	// Test free command
	stdout, stderr, code := RunPorts(t, "free", strconv.Itoa(port), "--force")
	if code != 0 {
		t.Fatalf("ports free failed: %s", stderr)
	}

	AssertOutputContains(t, stdout, "port "+strconv.Itoa(port)+" is free")

	// Verify port is actually free
	time.Sleep(500 * time.Millisecond)
	if fixture.isListening() {
		t.Errorf("port %d should be free after ports free", port)
	}
}

func TestKillCommand(t *testing.T) {
	port, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}

	fixture := NodeFixture(t, port, "kill-test")
	defer fixture.Stop()
	fixture.WaitReady(t)

	// Kill by the fixture's own project name — the tempdir's package.json
	// makes it a project no other listener shares. Matching the framework
	// instead ("Node.js") would stop every Node.js process on the machine.
	stdout, stderr, code := RunPorts(t, "kill", fixture.Project, "--force")
	if code != 0 {
		t.Fatalf("ports kill failed: %s", stderr)
	}

	AssertOutputContains(t, stdout, "stopped")

	// Verify port is actually free
	time.Sleep(500 * time.Millisecond)
	if fixture.isListening() {
		t.Errorf("port %d should be free after ports kill", port)
	}
}

func TestOutputFormats(t *testing.T) {
	port, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}

	fixture := HTTPFixture(t, port)
	defer fixture.Stop()
	fixture.WaitReady(t)

	// Test table output (default)
	stdout, stderr, code := RunPorts(t)
	if code != 0 {
		t.Fatalf("ports failed: %s", stderr)
	}
	AssertOutputContains(t, stdout, "PORT", "PROCESS", "PROJECT", "UPTIME")

	// Test JSON output
	stdout, stderr, code = RunPorts(t, "--json")
	if code != 0 {
		t.Fatalf("ports --json failed: %s", stderr)
	}

	var services []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &services); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(services) == 0 {
		t.Error("expected non-empty JSON output")
	}
}

func TestVersionFlag(t *testing.T) {
	stdout, stderr, code := RunPorts(t, "--version")
	if code != 0 {
		t.Fatalf("ports --version failed: %s", stderr)
	}
	AssertOutputContains(t, stdout, "ports")
}

func TestCompletionCommand(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		stdout, stderr, code := RunPorts(t, "completion", shell)
		if code != 0 {
			t.Fatalf("ports completion %s failed: %s", shell, stderr)
		}
		if len(stdout) == 0 {
			t.Errorf("expected completion output for %s", shell)
		}
	}
}

func TestWatchJSONStream(t *testing.T) {
	port, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}
	fixture := HTTPFixture(t, port)
	defer fixture.Stop()
	fixture.WaitReady(t)

	// 250ms ticks over a 3s window, then SIGINT — the graceful exit watch
	// promises. JSON mode emits one array per tick; even a 2s scan still
	// lands two of them inside the window.
	stdout, stderr, code := RunPortsInterrupted(t, 3*time.Second, "--watch", "--format", "json", "--interval", "250ms")
	if code != 0 {
		t.Fatalf("ports --watch exited %d: %s", code, stderr)
	}
	ticks := strings.Count(stdout, "\n]")
	if ticks < 2 {
		t.Errorf("expected at least 2 ticks, got %d in:\n%s", ticks, stdout)
	}
	if !strings.Contains(stdout, strconv.Itoa(port)) {
		t.Errorf("expected the fixture's port %d in the stream:\n%s", port, stdout)
	}
}

func TestWatchRejections(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"csv has no redraw story", []string{"--watch", "--format", "csv"}, "table or json"},
		{"tsv has no redraw story", []string{"--watch", "--format", "tsv"}, "table or json"},
		{"interval below minimum", []string{"--watch", "--format", "json", "--interval", "100ms"}, "--interval must be at least"},
		{"table in a pipe", []string{"--watch"}, "needs a terminal"},
		{"watch on kill", []string{"kill", "--watch", "no-such-service"}, "only applies to listing"},
		{"watch on free", []string{"free", "--watch", "1"}, "only applies to listing"},
		{"watch on find", []string{"find", "--watch", "no-such-service"}, "only applies to listing"},
		{"watch on doctor", []string{"doctor", "--watch", "1"}, "only applies to listing"},
		{"watch on diff", []string{"diff", "--watch", "before.json"}, "only applies to listing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, code := RunPorts(t, tt.args...)
			if code != 2 {
				t.Errorf("exit = %d, want 2 (stderr: %s)", code, stderr)
			}
			if !strings.Contains(stderr, tt.want) {
				t.Errorf("stderr %q missing %q", stderr, tt.want)
			}
			if stdout != "" {
				t.Errorf("expected no stdout, got %q", stdout)
			}
		})
	}
}

// Doctor on a port nothing can complain about: one listener, owned by
// the fixture, no stray sockets. The ResetFixture aborts its readiness
// probes with a RST so nothing lingers — any ⚠ here is a real finding.
func TestDoctorCleanPort(t *testing.T) {
	SkipIfWSL(t)
	port, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}
	fixture := ResetFixture(t, port)
	defer fixture.Stop()
	fixture.WaitReady(t)
	time.Sleep(200 * time.Millisecond) // let the last probe's RST settle

	stdout, stderr, code := RunPorts(t, "doctor", strconv.Itoa(port))
	if code != 0 {
		t.Fatalf("ports doctor on a healthy port exited %d: %s\n%s", code, stderr, stdout)
	}
	AssertOutputContains(t, stdout,
		":"+strconv.Itoa(port),
		"✓ listening on :"+strconv.Itoa(port),
		"nothing else holds")
	AssertOutputNotContains(t, stdout, "⚠", "✗")
}

// Doctor on a port whose listener closed first: this side lands in
// TIME_WAIT, the classic "free said it stopped but the port won't
// rebind" story. The fixture closes the instant it accepts, so a probe
// that lingers — rather than closing the moment the dial returns — makes
// the fixture's side the active closer, deterministically.
func TestDoctorTimeWaitWarns(t *testing.T) {
	port, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}
	fixture := TimeWaitFixture(t, port)
	defer fixture.Stop()
	fixture.WaitReady(t)

	// a held-open connection: the fixture's close lands first
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond) // the fixture accepts and closes while we hold
	conn.Close()

	stdout, _, code := RunPorts(t, "doctor", strconv.Itoa(port))
	if code != 1 {
		t.Fatalf("ports doctor with a TIME_WAIT finding exited %d, want 1:\n%s", code, stdout)
	}
	AssertOutputContains(t, stdout, "⚠", "TIME_WAIT", ":"+strconv.Itoa(port))
}

// Doctor on an unbound port names the absence instead of guessing.
func TestDoctorNothingListening(t *testing.T) {
	port, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}

	stdout, _, code := RunPorts(t, "doctor", strconv.Itoa(port))
	if code != 1 {
		t.Fatalf("ports doctor on an unbound port exited %d, want 1:\n%s", code, stdout)
	}
	AssertOutputContains(t, stdout, "✗ nothing is listening on :"+strconv.Itoa(port))
}

// Bad arguments are usage errors, not diagnoses.
func TestDoctorBadPort(t *testing.T) {
	stdout, stderr, code := RunPorts(t, "doctor", "not-a-port")
	if code != 2 {
		t.Fatalf("ports doctor not-a-port exited %d, want 2", code)
	}
	AssertOutputContains(t, stderr, "invalid port")
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
}

// JSON parity: the same findings, as an array of check/status pairs.
func TestDoctorJSON(t *testing.T) {
	SkipIfWSL(t)
	port, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}
	fixture := ResetFixture(t, port)
	defer fixture.Stop()
	fixture.WaitReady(t)
	time.Sleep(200 * time.Millisecond)

	stdout, _, code := RunPorts(t, "doctor", strconv.Itoa(port), "--format", "json")
	if code != 0 {
		t.Fatalf("ports doctor --format json exited %d:\n%s", code, stdout)
	}
	var findings []map[string]any
	if err := json.Unmarshal([]byte(stdout), &findings); err != nil {
		t.Fatalf("doctor json does not parse: %v\n%s", err, stdout)
	}
	if len(findings) == 0 {
		t.Fatalf("doctor json is empty:\n%s", stdout)
	}
	for _, f := range findings {
		if _, ok := f["check"]; !ok {
			t.Errorf("finding %v has no check field", f)
		}
		if _, ok := f["status"]; !ok {
			t.Errorf("finding %v has no status field", f)
		}
	}
}

// Bare doctor runs the machine-wide sweep. What it finds depends on the
// machine — shared CI runners have other users' listeners — so only the
// shape is asserted: findings, and never a usage error.
func TestDoctorMachine(t *testing.T) {
	stdout, stderr, code := RunPorts(t, "doctor")
	if code == 2 {
		t.Fatalf("bare ports doctor is a usage error: %s", stderr)
	}
	if stdout == "" {
		t.Fatalf("bare ports doctor printed nothing")
	}
	first, _ := utf8.DecodeRuneInString(stdout)
	if !strings.ContainsRune("✓⚠✗", first) {
		t.Errorf("findings should start with a status prefix:\n%s", stdout)
	}
}

// The pointer, not the marketing: a failed free names doctor as the
// next step.
func TestFreePointsAtDoctor(t *testing.T) {
	port, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}

	_, stderr, code := RunPorts(t, "free", strconv.Itoa(port))
	if code != 1 {
		t.Fatalf("ports free on an unbound port exited %d, want 1", code)
	}
	AssertOutputContains(t, stderr, "ports doctor "+strconv.Itoa(port))
}

// Doctor hands over the URL only once it verified the port speaks HTTP —
// the honest home for a clickable localhost link. Only the finding is
// asserted: on WSL the mirror warn correctly makes the exit nonzero.
func TestDoctorHTTPProbe(t *testing.T) {
	port, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}

	fixture := HTTPFixture(t, port)
	defer fixture.Stop()
	fixture.WaitReady(t)

	stdout, _, code := RunPorts(t, "doctor", strconv.Itoa(port))
	if code == 2 {
		t.Fatalf("doctor on a healthy HTTP port is a usage error: %s", stdout)
	}
	AssertOutputContains(t, stdout, "responds to HTTP → http://localhost:"+strconv.Itoa(port))
}

// The URL finding is earned: a listener that only accepts and closes —
// no HTTP in sight — gets no clickable claim.
func TestDoctorHTTPProbeSilent(t *testing.T) {
	port, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}

	fixture := TimeWaitFixture(t, port)
	defer fixture.Stop()
	fixture.WaitReady(t)

	stdout, _, _ := RunPorts(t, "doctor", strconv.Itoa(port))
	AssertOutputNotContains(t, stdout, "responds to HTTP")
}

// The snapshot round trip: `--format snapshot` captures the moment,
// `diff` reads it back — first against an unchanged machine, then after
// the world moves (a stop, a start, a restart), and in both formats.
func TestDiffCommand(t *testing.T) {
	a, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}
	b, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}

	first := HTTPFixture(t, a)
	defer first.Stop()
	first.WaitReady(t)

	// save the moment — the envelope carries the version and the rows
	snap, _, code := RunPorts(t, "--format", "snapshot", "--port", strconv.Itoa(a))
	if code != 0 {
		t.Fatalf("ports --format snapshot exited %d:\n%s", code, snap)
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(snap), &envelope); err != nil {
		t.Fatalf("snapshot does not parse: %v\n%s", err, snap)
	}
	if v, ok := envelope["version"].(float64); !ok || v != 1 {
		t.Errorf("snapshot version = %v, want 1:\n%s", envelope["version"], snap)
	}
	if _, ok := envelope["generated_at"]; !ok {
		t.Errorf("snapshot has no generated_at:\n%s", snap)
	}
	if services, ok := envelope["services"].([]any); !ok || len(services) != 1 {
		t.Errorf("snapshot services = %v, want the one row:\n%s", envelope["services"], snap)
	}
	path := filepath.Join(t.TempDir(), "before.json")
	if err := os.WriteFile(path, []byte(snap), 0o644); err != nil {
		t.Fatal(err)
	}

	// an unchanged machine: exit 0, the sentence that says so
	stdout, stderr, code := RunPorts(t, "diff", path, "--port", strconv.Itoa(a))
	if code != 0 {
		t.Fatalf("diff against an unchanged machine exited %d: %s\n%s", code, stderr, stdout)
	}
	AssertOutputContains(t, stdout, "no changes")

	// a plain --format json capture loads too — same rows, no envelope
	bare, _, code := RunPorts(t, "--format", "json", "--port", strconv.Itoa(a))
	if code != 0 {
		t.Fatalf("ports --format json exited %d", code)
	}
	barePath := filepath.Join(t.TempDir(), "bare.json")
	if err := os.WriteFile(barePath, []byte(bare), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, _, code = RunPorts(t, "diff", barePath, "--port", strconv.Itoa(a))
	if code != 0 {
		t.Fatalf("diff against a bare json capture exited %d:\n%s", code, stdout)
	}
	AssertOutputContains(t, stdout, "no changes")

	// the world moves: the listener stops, another starts
	first.Stop()
	second := HTTPFixture(t, b)
	defer second.Stop()
	second.WaitReady(t)

	stdout, stderr, code = RunPorts(t, "diff", path, "--port", fmt.Sprintf("%d,%d", a, b))
	if code != 1 {
		t.Fatalf("diff after the world moved exited %d, want 1:\n%s%s", code, stderr, stdout)
	}
	// the new listener is the stable story on every machine; the stopped
	// one is "gone" on plain Linux, but WSL's mirror keeps the port
	// visible as a windows-side row — an honest "changed" there
	AssertOutputContains(t, stdout,
		"new", ":"+strconv.Itoa(b), "listening now — not in the snapshot")
	AssertOutputContains(t, stdout, ":"+strconv.Itoa(a))
	if !onWSL() {
		AssertOutputContains(t, stdout, "gone", "in the snapshot, no longer listening")
	}

	// json parity: the same changes as an array of port/change pairs
	stdout, _, code = RunPorts(t, "diff", path, "--format", "json", "--port", fmt.Sprintf("%d,%d", a, b))
	if code != 1 {
		t.Fatalf("diff --format json after the world moved exited %d, want 1:\n%s", code, stdout)
	}
	var diffs []map[string]any
	if err := json.Unmarshal([]byte(stdout), &diffs); err != nil {
		t.Fatalf("diff json does not parse: %v\n%s", err, stdout)
	}
	verbs := map[float64]string{}
	for _, d := range diffs {
		if change, ok := d["change"].(string); ok {
			if port, ok := d["port"].(float64); ok {
				verbs[port] = change
			}
		}
	}
	if verbs[float64(a)] != "gone" && !(onWSL() && verbs[float64(a)] == "changed") {
		t.Errorf("diff json verb for the stopped port = %q, want gone (changed only under the WSL mirror):\n%s", verbs[float64(a)], stdout)
	}
	if verbs[float64(b)] != "new" {
		t.Errorf("diff json verb for the fresh port = %q, want new:\n%s", verbs[float64(b)], stdout)
	}

	// a restart on the same port: changed — the mirror makes the exact
	// wording machine-dependent, so only the verb is asserted here; the
	// message shapes are locked by the unit tests
	third := HTTPFixture(t, a)
	defer third.Stop()
	third.WaitReady(t)
	stdout, _, code = RunPorts(t, "diff", path, "--port", strconv.Itoa(a))
	if code != 1 {
		t.Fatalf("diff after a restart exited %d, want 1:\n%s", code, stdout)
	}
	AssertOutputContains(t, stdout, "changed", ":"+strconv.Itoa(a))

	// a filter on diff scopes the whole comparison: the restarted port,
	// outside the range, is left out — never reported gone
	stdout, _, code = RunPorts(t, "diff", path, "--port", strconv.Itoa(b))
	if code != 1 {
		t.Fatalf("diff scoped to the fresh port exited %d, want 1:\n%s", code, stdout)
	}
	AssertOutputContains(t, stdout, "new", ":"+strconv.Itoa(b))
	AssertOutputNotContains(t, stdout, ":"+strconv.Itoa(a))

	// the machine settles: a fresh snapshot, diffed clean, renders an
	// empty array in json — never null — and exits 0
	settled, _, code := RunPorts(t, "--format", "snapshot", "--port", fmt.Sprintf("%d,%d", a, b))
	if code != 0 {
		t.Fatalf("ports --format snapshot exited %d", code)
	}
	settledPath := filepath.Join(t.TempDir(), "after.json")
	if err := os.WriteFile(settledPath, []byte(settled), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, _, code = RunPorts(t, "diff", settledPath, "--format", "json", "--port", fmt.Sprintf("%d,%d", a, b))
	if code != 0 {
		t.Fatalf("diff against a fresh snapshot exited %d:\n%s", code, stdout)
	}
	if trimmed := strings.TrimSpace(stdout); trimmed != "[]" {
		t.Errorf("clean diff json = %q, want []", trimmed)
	}

	// diffs have no csv story — the format is rejected, not guessed at
	_, stderr, code = RunPorts(t, "diff", settledPath, "--format", "csv", "--port", strconv.Itoa(a))
	if code != 2 {
		t.Errorf("diff --format csv exited %d, want 2", code)
	}
	AssertOutputContains(t, stderr, "table or json")
}

// firstPortColumn reads the csv body (header off) and returns the port of
// each row, in display order.
func firstPortColumn(t *testing.T, csvOut string) []int {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(csvOut), "\n")
	if len(lines) < 2 {
		t.Fatalf("no csv rows in %q", csvOut)
	}
	var ports []int
	for _, line := range lines[1:] {
		port, err := strconv.Atoi(strings.SplitN(line, ",", 2)[0])
		if err != nil {
			t.Fatalf("bad csv row %q: %v", line, err)
		}
		ports = append(ports, port)
	}
	return ports
}

func TestSortFlag(t *testing.T) {
	hi, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}
	lo, err := FindFreePort()
	if err != nil {
		t.Fatal(err)
	}
	if lo > hi { // the two modes must disagree about which row is first
		hi, lo = lo, hi
	}

	// the older listener starts first, so uptime and port disagree
	old := HTTPFixture(t, hi)
	defer old.Stop()
	old.WaitReady(t)
	time.Sleep(1100 * time.Millisecond)
	new := HTTPFixture(t, lo)
	defer new.Stop()
	new.WaitReady(t)

	// both runs filter down to exactly the two fixture rows
	filter := []string{"--format", "csv", "--port", fmt.Sprintf("%d,%d", hi, lo)}

	out, _, code := RunPorts(t, append([]string{"--sort", "port"}, filter...)...)
	if code != 0 {
		t.Fatalf("ports --sort port exited %d", code)
	}
	if got := firstPortColumn(t, out); len(got) == 2 && got[0] != lo {
		t.Errorf("--sort port = %v, want %d first", got, lo)
	}

	out, _, code = RunPorts(t, append([]string{"--sort", "uptime"}, filter...)...)
	if code != 0 {
		t.Fatalf("ports --sort uptime exited %d", code)
	}
	if got := firstPortColumn(t, out); len(got) == 2 && got[0] != hi {
		t.Errorf("--sort uptime = %v, want the older listener (%d) first", got, hi)
	}

	_, stderr, code := RunPorts(t, "--sort", "name")
	if code != 2 {
		t.Fatalf("ports --sort name exited %d, want 2", code)
	}
	AssertOutputContains(t, stderr, "invalid sort")
}

// The exit-code grammar, uniform across commands: 0 clean, 1 nothing
// matched or something wrong (grep-style), 2 the invocation itself was
// unusable. Every row here is a bad invocation that once exited 1.
func TestExitCodes(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"unknown command exits 2", []string{"frobnicate"}, 2},
		{"unknown flag exits 2", []string{"--badflag"}, 2},
		{"unknown flag on a subcommand exits 2", []string{"kill", "--badflag", "shop"}, 2},
		{"flag missing its value exits 2", []string{"--format"}, 2},
		{"kill with no name exits 2", []string{"kill"}, 2},
		{"find with no name exits 2", []string{"find"}, 2},
		{"free with no port exits 2", []string{"free"}, 2},
		{"unknown format exits 2", []string{"--format", "yaml"}, 2},
		{"invalid completion shell exits 2", []string{"completion", "badshell"}, 2},
		{"doctor with a non-numeric port exits 2", []string{"doctor", "99b"}, 2},
		{"diff with no snapshot exits 2", []string{"diff"}, 2},
		{"diff with a missing snapshot exits 2", []string{"diff", "/no/such/before.json"}, 2},
		{"diff with a second file exits 2", []string{"diff", "a.json", "b.json"}, 2},
		{"invalid sort exits 2", []string{"--sort", "name"}, 2},
		{"find with no match exits 1 grep-style", []string{"find", "zzz-nothing-xyz"}, 1},
		{"bare list exits 0", []string{"--json"}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stdout, stderr, code := RunPorts(t, c.args...)
			if code != c.want {
				t.Errorf("ports %s exited %d, want %d (stderr: %s)", strings.Join(c.args, " "), code, c.want, stderr)
			}
			// errors report on stderr; stdout stays clean for scripting
			if c.want != 0 && stdout != "" {
				t.Errorf("expected no stdout for %v, got %q", c.args, stdout)
			}
		})
	}
}
