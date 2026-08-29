package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
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

	// kill uses exact matching on framework name
	stdout, stderr, code := RunPorts(t, "kill", "Node.js", "--force")
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
