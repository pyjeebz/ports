package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	gproc "github.com/shirou/gopsutil/v4/process"
	"golang.org/x/term"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/pyjeebz/ports/internal/router"
	"github.com/pyjeebz/ports/internal/service"
	"github.com/spf13/cobra"
)

var proxyStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Run the router in the background, started at login",
	Long: `ports proxy start keeps the router running in the background and starts
it again at login: a systemd user unit on Linux, a launchd agent on
macOS. Elsewhere — Windows, WSL without systemd — it runs as a detached
process until logout or ports proxy stop.

On Linux, port 80 needs a one-time capability; in a terminal, start
offers to set it (sudo setcap), so URLs need no port. Say no and the
router uses 8480.

Starting a router that already runs changes nothing and exits 0.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error { return proxyStart() },
}

var proxyStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the router and remove its background service",
	Long: `ports proxy stop stops the router — a background one, or one running
in another terminal — and removes what ports proxy start set up, so it
doesn't come back at login. Names stop resolving; dev servers keep
running untouched. Stopping when no router runs exits 0.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error { return proxyStop() },
}

var proxyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Whether the router runs, where, and how it's kept running",
	Long: `ports proxy status says whether a router runs, on which port, and how
it's kept running.

Exit codes: 0 when a router runs, 1 when none does (so
` + "`ports proxy status || ports proxy start`" + ` works), 2 for a bad
invocation. --json prints {"running", "pid", "port", "managed_by",
"starts_at_login", "log"}.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error { return proxyStatus() },
}

func init() {
	proxyStatusCmd.Flags().BoolVar(&jsonOut, "json", false, "JSON output")
	proxyCmd.AddCommand(proxyStartCmd, proxyStopCmd, proxyStatusCmd)
}

func proxyStart() error {
	if info, ok := router.Running(); ok {
		how := "in another terminal"
		if kind, installed := service.Installed(); installed {
			how = "as a " + string(kind) + " service"
		}
		fmt.Println(inspect.Mark(fmt.Sprintf("✓ the router already runs %s, on :%d", how, info.Port), false))
		return nil
	}
	if err := refuseSudo(); err != nil {
		return err
	}
	offerPort80()

	exe, err := service.Executable()
	if err != nil {
		return err
	}
	kind := service.Available()
	if err := service.Install(kind, exe, stateEnv()); err != nil {
		if kind == service.Process {
			return fmt.Errorf("can't start the router: %w", err)
		}
		// no login session to hang a service on — a Mac reached over
		// SSH, a Linux box without a user manager: run it detached
		fmt.Fprintf(os.Stderr, "  couldn't set up the %s service (%v) — running the router in the background until logout instead\n", kind, err)
		_ = service.Uninstall(kind)
		kind = service.Process
		if err := service.Install(kind, exe, stateEnv()); err != nil {
			return fmt.Errorf("can't start the router: %w", err)
		}
	}
	info, ok := waitRouter(5 * time.Second)
	if !ok {
		return exitError{code: 1, msg: "the router didn't come up — " + logHint(kind)}
	}
	fmt.Println(inspect.Mark(fmt.Sprintf("✓ routing *.localhost on :%d — %s", info.Port, describeKind(kind)), false))
	if info.Port != router.DefaultPort {
		fmt.Fprintf(os.Stderr, "  names carry :%d — %s\n", info.Port, "`ports doctor` explains how to get :80")
	}
	return nil
}

func describeKind(kind service.Kind) string {
	switch kind {
	case service.Systemd:
		return "a systemd user unit, started at login"
	case service.Launchd:
		return "a launchd agent, started at login"
	default:
		return "in the background until logout (no login start here)"
	}
}

func logHint(kind service.Kind) string {
	if kind == service.Systemd {
		return "`journalctl --user -u ports-proxy` shows why"
	}
	return "the log is " + inspect.ShortPath(service.LogPath())
}

// stateEnv is the environment a background router needs to share this
// shell's state directory — set only when it was moved from the default.
func stateEnv() []string {
	var env []string
	for _, k := range []string{"PORTS_STATE_DIR", "XDG_STATE_HOME"} {
		if v := os.Getenv(k); v != "" {
			env = append(env, k+"="+v)
		}
	}
	return env
}

func waitRouter(within time.Duration) (router.Info, bool) {
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if info, ok := router.Running(); ok {
			return info, true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return router.Info{}, false
}

// offerPort80 asks, in a terminal on Linux, whether to give ports the
// capability to listen on 80 — once, with the exact command shown. Off a
// terminal, or answered no, the router takes the fallback port.
func offerPort80() {
	if runtime.GOOS != "linux" || !term.IsTerminal(int(os.Stdin.Fd())) {
		return
	}
	ls, err := router.Listen(router.DefaultPort)
	if err == nil {
		for _, l := range ls {
			_ = l.Close()
		}
		return // 80 is ours to take already
	}
	if !errors.Is(err, os.ErrPermission) {
		return // held by something else: no capability helps
	}
	exe, err := service.Executable()
	if err != nil {
		return
	}
	fmt.Fprintf(os.Stderr, "Port 80 needs a capability on Linux, so URLs can be http://shop.localhost with no port.\n")
	fmt.Fprintf(os.Stderr, "Run `sudo setcap cap_net_bind_service=+ep %s` now? [y/N] ", exe)
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
		fmt.Fprintf(os.Stderr, "  no change — the router uses :%d\n", router.FallbackPort)
		return
	}
	real, err := realPath(exe)
	if err != nil {
		real = exe
	}
	setcap := exec.Command("sudo", "setcap", "cap_net_bind_service=+ep", real)
	setcap.Stdin, setcap.Stdout, setcap.Stderr = os.Stdin, os.Stderr, os.Stderr
	if err := setcap.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "  setcap didn't run (%v) — the router uses :%d\n", err, router.FallbackPort)
	}
}

func proxyStop() error {
	stopped := false
	if kind, installed := service.Installed(); installed {
		if err := service.Uninstall(kind); err != nil {
			return fmt.Errorf("removing the %s service: %w", kind, err)
		}
		stopped = true
	}
	// a router the service didn't own — detached, or in another terminal
	if info, ok := router.Running(); ok {
		if err := stopRouterPid(info.Pid); err != nil {
			return err
		}
		stopped = true
	}
	if !stopped {
		fmt.Println("no router was running")
		return nil
	}
	if _, ok := router.Running(); ok {
		return exitError{code: 1, msg: "the router is still running"}
	}
	fmt.Println(inspect.Mark("✓ router stopped — names are gone; dev servers keep running", false))
	return nil
}

// stopRouterPid stops the process recorded as the router — after
// checking it is a ports router, so a recycled pid is never signalled.
func stopRouterPid(pid int) error {
	p, err := gproc.NewProcess(int32(pid))
	if err != nil {
		return nil // already gone
	}
	args, _ := p.CmdlineSlice()
	if len(args) < 2 || args[1] != "proxy" {
		return fmt.Errorf("pid %d isn't a ports router (%s) — not stopping it", pid, strings.Join(args, " "))
	}
	if err := p.Terminate(); err != nil {
		return err
	}
	for range 30 {
		if alive, _ := p.IsRunning(); !alive {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("the router (pid %d) didn't stop", pid)
}

func proxyStatus() error {
	info, running := router.Running()
	kind, installed := service.Installed()
	managed := "none"
	switch {
	case installed:
		managed = string(kind)
	case running:
		managed = "process"
	}
	log := ""
	switch {
	case installed && kind == service.Systemd:
		log = "journalctl --user -u ports-proxy"
	case installed:
		log = service.LogPath()
	case running:
		// a detached router writes the log file; one in a terminal
		// writes to the terminal — the file, if any, is older than it
		if st, err := os.Stat(service.LogPath()); err == nil && !st.ModTime().Before(info.Started.Add(-time.Second)) {
			log = service.LogPath()
		}
	}
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(struct {
			Running       bool   `json:"running"`
			Pid           int    `json:"pid,omitempty"`
			Port          int    `json:"port,omitempty"`
			ManagedBy     string `json:"managed_by"`
			StartsAtLogin bool   `json:"starts_at_login"`
			Log           string `json:"log,omitempty"`
		}{running, info.Pid, info.Port, managed, installed, log}); err != nil {
			return err
		}
	} else {
		switch {
		case running:
			fmt.Println(inspect.Mark(fmt.Sprintf("✓ routing *.localhost on :%d (pid %d)", info.Port, info.Pid), false))
			if installed {
				fmt.Printf("  %s\n", describeKind(kind))
			} else {
				fmt.Println("  not started at login — `ports proxy start` sets that up")
			}
			if log != "" {
				fmt.Printf("  log: %s\n", inspect.ShortPath(log))
			}
		case installed:
			fmt.Printf("✗ not running, though a %s service is set up — %s\n", kind, logHint(kind))
		default:
			fmt.Println("not running — `ports proxy start` runs it in the background")
		}
	}
	if !running {
		return exitError{code: 1}
	}
	return nil
}

// realPath is the file a path leads to: setcap marks the file, not a
// link to it.
func realPath(p string) (string, error) {
	return filepath.EvalSymlinks(p)
}
