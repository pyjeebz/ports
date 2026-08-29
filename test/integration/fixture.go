package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Fixture represents a test process that listens on a port.
type Fixture struct {
	Name     string
	Port     int
	Cmd      *exec.Cmd
	Cancel   context.CancelFunc
	Ready    chan struct{}
	mu       sync.Mutex
	StartErr error
}

// NewFixture creates a new test fixture.
func NewFixture(t *testing.T, name string, port int, command string, args ...string) *Fixture {
	ctx, cancel := context.WithCancel(context.Background())
	f := &Fixture{
		Name:   name,
		Port:   port,
		Cmd:    exec.CommandContext(ctx, command, args...),
		Cancel: cancel,
		Ready:  make(chan struct{}),
	}

	f.Cmd.Stdout = os.Stdout
	f.Cmd.Stderr = os.Stderr

	if err := f.Cmd.Start(); err != nil {
		t.Fatalf("failed to start %s: %v", name, err)
	}

	go f.waitReady(t)
	return f
}

// waitReady waits for the process to start listening on the port.
func (f *Fixture) waitReady(t *testing.T) {
	f.mu.Lock()
	defer f.mu.Unlock()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if f.isListening() {
			close(f.Ready)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	f.StartErr = fmt.Errorf("timeout waiting for %s to listen on port %d", f.Name, f.Port)
	if f.Ready != nil {
		close(f.Ready)
	}
}

func (f *Fixture) isListening() bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", f.Port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// WaitReady blocks until the fixture is ready or times out.
func (f *Fixture) WaitReady(t *testing.T) {
	select {
	case <-f.Ready:
		if f.StartErr != nil {
			t.Fatal(f.StartErr)
		}
	case <-time.After(15 * time.Second):
		t.Fatalf("timeout waiting for %s", f.Name)
	}
}

// Stop terminates the fixture process.
func (f *Fixture) Stop() {
	if f.Cancel != nil {
		f.Cancel()
	}
	if f.Cmd != nil && f.Cmd.Process != nil {
		f.Cmd.Process.Kill()
		f.Cmd.Wait()
	}
}

// HTTPFixture creates a simple HTTP server fixture.
func HTTPFixture(t *testing.T, port int) *Fixture {
	dir := t.TempDir()
	indexHTML := filepath.Join(dir, "index.html")
	if err := os.WriteFile(indexHTML, []byte("<html><body>Test</body></html>"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := "python3"
	args := []string{"-m", "http.server", strconv.Itoa(port)}
	if runtime.GOOS == "windows" {
		cmd = "python"
		args = []string{"-m", "http.server", strconv.Itoa(port)}
	}

	f := NewFixture(t, "http", port, cmd, args...)
	f.Cmd.Dir = dir
	return f
}

// TCPFixture creates a simple TCP listener fixture.
func TCPFixture(t *testing.T, port int) *Fixture {
	return NewFixture(t, "tcp", port, "nc", "-l", strconv.Itoa(port))
}

// NodeFixture creates a Node.js process fixture.
func NodeFixture(t *testing.T, port int, script string) *Fixture {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "server.js")
	content := fmt.Sprintf(`
const http = require('http');
const server = http.createServer((req, res) => {
  res.writeHead(200);
  res.end('Hello from %s');
});
server.listen(%d, () => {
  console.log('Listening on port %d');
});
`, script, port, port)
	if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	return NewFixture(t, "node-"+script, port, "node", scriptPath)
}

// GoFixture creates a Go process fixture.
func GoFixture(t *testing.T, port int, name string) *Fixture {
	dir := t.TempDir()
	mainGo := filepath.Join(dir, "main.go")
	content := fmt.Sprintf(`package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from %s")
	})
	fmt.Printf("Listening on port %%d\n", %d)
	http.ListenAndServe(":%d", nil)
}
`, name, port, port)
	if err := os.WriteFile(mainGo, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	goMod := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.23\n"), 0644); err != nil {
		t.Fatal(err)
	}

	return NewFixture(t, "go-"+name, port, "go", "run", "main.go")
}

// FindFreePort finds an available TCP port.
func FindFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// RunPorts runs the ports binary and returns stdout, stderr, and exit code.
func RunPorts(t *testing.T, args ...string) (string, string, int) {
	binary := os.Getenv("PORTS_BINARY")
	if binary == "" {
		binary = filepath.Join(os.TempDir(), "ports-integration")
	}
	cmd := exec.Command(binary, args...)
	cmd.Dir, _ = os.Getwd()

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return stdout.String(), stderr.String(), exitCode
}

// AssertOutputContains checks if output contains expected strings.
func AssertOutputContains(t *testing.T, output string, expected ...string) {
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected output to contain %q, got:\n%s", exp, output)
		}
	}
}

// AssertOutputNotContains checks if output does not contain unexpected strings.
func AssertOutputNotContains(t *testing.T, output string, unexpected ...string) {
	for _, unexp := range unexpected {
		if strings.Contains(output, unexp) {
			t.Errorf("expected output not to contain %q, got:\n%s", unexp, output)
		}
	}
}
