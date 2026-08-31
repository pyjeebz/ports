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

// ResetFixture listens and aborts every accepted connection with a RST
// (SO_LINGER 0). Readiness probes therefore leave nothing in the socket
// table — no TIME_WAIT, no CLOSE_WAIT — so a doctor run on this port
// sees only the listener and reads clean. Use it for deterministic
// all-✓ doctor tests.
func ResetFixture(t *testing.T, port int) *Fixture {
	return socketFixture(t, "reset", port, "c.setsockopt(socket.SOL_SOCKET, socket.SO_LINGER, struct.pack('ii', 1, 0))")
}

// TimeWaitFixture listens and closes first on every accepted connection,
// so its side of each connection lands in TIME_WAIT — the state that
// makes a port look held after its listener is gone. Use it for
// deterministic doctor warn tests.
func TimeWaitFixture(t *testing.T, port int) *Fixture {
	return socketFixture(t, "timewait", port, "")
}

// socketFixture runs a python listener whose accept loop closes every
// connection, with an optional extra close-time socket option.
func socketFixture(t *testing.T, name string, port int, onClose string) *Fixture {
	dir := t.TempDir()
	script := fmt.Sprintf(`import socket, struct

s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(("", %d))
s.listen(64)
while True:
    c, _ = s.accept()
    %s
    c.close()
`, port, onClose)
	scriptPath := filepath.Join(dir, "server.py")
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := "python3"
	if runtime.GOOS == "windows" {
		cmd = "python"
	}
	return NewFixture(t, "socket-"+name, port, cmd, scriptPath)
}

// onWSL reports whether the test is running inside WSL, where mirrored
// networking forwards a Linux listener to the Windows side — doctor's
// mirror warn is correct there, so exit-0 assertions can't hold.
func onWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	b, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	s := strings.ToLower(string(b))
	return strings.Contains(s, "microsoft") || strings.Contains(s, "wsl")
}

// SkipIfWSL explains why a clean-port doctor test can't assert exit 0 on
// this machine.
func SkipIfWSL(t *testing.T) {
	t.Helper()
	if onWSL() {
		t.Skip("WSL mirrors the listener into Windows — the mirror warn is correct here")
	}
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
	cmd := exec.Command(portsBinaryPath(), args...)
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

func portsBinaryPath() string {
	if binary := os.Getenv("PORTS_BINARY"); binary != "" {
		return binary
	}
	return filepath.Join(os.TempDir(), "ports-integration")
}

// RunPortsInterrupted starts the binary, lets it run for the given window,
// then sends SIGINT — the graceful path watch promises — and returns what
// it printed. If it ignores the signal, it is killed and the test sees
// a signal exit rather than hanging.
func RunPortsInterrupted(t *testing.T, window time.Duration, args ...string) (string, string, int) {
	cmd := exec.Command(portsBinaryPath(), args...)
	cmd.Dir, _ = os.Getwd()

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(window)
	_ = cmd.Process.Signal(os.Interrupt)

	// grace period: a watch mid-scan finishes the scan, then exits
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		exitCode := 0
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if err != nil {
			exitCode = 1
		}
		return stdout.String(), stderr.String(), exitCode
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		return stdout.String(), stderr.String(), -1
	}
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
