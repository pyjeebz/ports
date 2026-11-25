package cmd

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"

	gproc "github.com/shirou/gopsutil/v4/process"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/pyjeebz/ports/internal/naming"
	"github.com/pyjeebz/ports/internal/router"
	"github.com/pyjeebz/ports/internal/routes"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var runPort int

var runCmd = &cobra.Command{
	Use:   "run [name] -- <command>",
	Short: "Start a dev server on a free port, under its name",
	Long: `ports run starts a dev server on a port ports picks, and gives it its
name for as long as it runs:

  $ ports run -- npm run dev
  → http://shop.localhost (:4123)

The port comes from 4000-4999 — the same one each time when it's free —
and reaches the server the way its framework listens for one: PORT for
Next.js, Nuxt and most servers, --port (and --strictPort) for Vite,
Astro and Angular. A command that names its own --port keeps it.

The name is the project's, the way ports names a server it finds: shop,
api.shop in apps/api, ui.shop in a worktree on fix/ui. Give one to
choose: ports run docs -- npm run docs.

When the project is already running, ports run says where instead of
starting a second copy — the duplicate-dev-server trap. Name the new
one to run both.

ports run starts only the command given and supervises nothing: no
restarts, no logs. The name goes when the command exits, and ports run
exits with the command's code.`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dash := cmd.ArgsLenAtDash()
		if dash < 0 || dash == len(args) {
			return exitError{code: 2, msg: "give the command after --: ports run -- npm run dev"}
		}
		if dash > 1 {
			return exitError{code: 2, msg: fmt.Sprintf("one name before --, got %s", strings.Join(args[:dash], " "))}
		}
		if cmd.Flags().Changed("port") && (runPort < 1 || runPort > 65535) {
			return exitError{code: 2, msg: fmt.Sprintf("--port %d is out of range (1-65535)", runPort)}
		}
		var name string
		if dash == 1 {
			name = nameArg(args[0])
			if !routes.ValidName(name) {
				return exitError{code: 2, msg: fmt.Sprintf("%q can't be a name — use letters, digits and hyphens, with dots between parts (api.shop)", args[0])}
			}
		}
		return runCommand(name, args[dash:])
	},
}

func init() {
	runCmd.Flags().IntVar(&runPort, "port", 0, "Run on this port instead of a free one from 4000-4999")
	rootCmd.AddCommand(runCmd)
}

func runCommand(name string, command []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	dir = inspect.RealPath(dir)
	cfg := buildInspectConfig()
	root := inspect.ProjectRoot(cfg, dir)

	script := scriptText(command, dir)
	framework := runFramework(script)
	if name == "" {
		name = naming.Names([]naming.App{{Project: root, Cwd: dir, Framework: framework}}, naming.LoadClaims())[0]
		if name == "" {
			name = naming.Label(filepath.Base(dir))
		}
		if name == "" {
			return exitError{code: 2, msg: "this folder gives no name — pass one: ports run <name> -- …"}
		}
	}

	services, err := scanNamed(cfg)
	if err != nil {
		return err
	}
	if msg := nameTaken(name, services); msg != "" {
		return exitError{code: 1, msg: msg}
	}

	port := uint32(runPort)
	if own, ok := ownPort(command); ok {
		port = own
	}
	if port == 0 {
		if port, err = pickPort(name); err != nil {
			return err
		}
	}

	started, err := routes.ProcessStarted(int32(os.Getpid()))
	if err != nil {
		return err
	}
	argv := withPortFlags(command, framework, port)
	reg := routes.Run{Name: name, Port: port, Pid: int32(os.Getpid()), Started: started, Dir: root, Command: strings.Join(argv, " ")}
	if err := routes.Register(reg); err != nil {
		return fmt.Errorf("can't register %s: %w", name, err)
	}
	defer func() { _ = routes.Unregister(reg.Pid) }()

	child := exec.Command(argv[0], argv[1:]...)
	child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
	child.Env = append(os.Environ(), portEnv(port)...)
	if err := child.Start(); err != nil {
		var ee *exec.Error
		if errors.As(err, &ee) {
			return exitError{code: 127, msg: fmt.Sprintf("%s: command not found", argv[0])}
		}
		return err
	}
	if info, ok := router.Running(); ok {
		fmt.Fprintf(os.Stderr, "→ %s (:%d)\n", router.URL(name, info.Port), port)
	} else {
		fmt.Fprintf(os.Stderr, "→ http://localhost:%d — `ports proxy` serves it at http://%s.localhost\n", port, name)
	}
	stop := forwardSignals(child.Process)
	err = child.Wait()
	stop()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code := exit.ExitCode()
			if code < 0 {
				code = 1 // ended by a signal
			}
			return exitError{code: code}
		}
		return err
	}
	return nil
}

// nameTaken explains why name can't be had right now, or is "".
func nameTaken(name string, services []inspect.Service) string {
	for _, r := range routes.LiveRuns() {
		if r.Name == name {
			return fmt.Sprintf("%s is taken by another ports run (pid %d, :%d) — name this one: ports run %s-2 -- …", name, r.Pid, r.Port, name)
		}
	}
	if aliases, err := routes.LoadAliases(); err == nil {
		if port, ok := aliases[name]; ok {
			return fmt.Sprintf("%s is an alias for :%d — name this one, or drop the alias: ports alias %s --remove", name, port, name)
		}
	}
	for _, s := range services {
		if s.Name == name {
			return fmt.Sprintf("%s is already running at %s (%s, :%d) — use it, or name this one to run both: ports run %s-2 -- …",
				name, cmp.Or(s.NameURL, s.URL), inspect.ProcessName(s), s.Port, name)
		}
	}
	return ""
}

// pickPort picks a port from 4000-4999 that nothing holds and no other
// ports run was handed, starting from one the name hashes to — so a
// project tends to land on the same port each time, and a bookmark of
// localhost:4123 keeps working too.
func pickPort(name string) (uint32, error) {
	taken := map[uint32]bool{}
	for _, r := range routes.LiveRuns() {
		taken[r.Port] = true
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	start := h.Sum32() % 1000
	for i := range uint32(1000) {
		port := 4000 + (start+i)%1000
		if !taken[port] && bindable(port) {
			return port, nil
		}
	}
	return 0, errors.New("no free port in 4000-4999")
}

// bindable reports whether a server could listen on port now, on either
// loopback and the wildcard.
func bindable(port uint32) bool {
	for _, addr := range []string{"127.0.0.1", "0.0.0.0", "::1"} {
		l, err := net.Listen("tcp", net.JoinHostPort(addr, strconv.Itoa(int(port))))
		if err != nil {
			if addr == "::1" && !strings.Contains(err.Error(), "in use") {
				continue // no IPv6 loopback here
			}
			return false
		}
		_ = l.Close()
	}
	return true
}

// portEnv is how most servers are told their port: PORT, read by
// Next.js, Nuxt, Create React App, Rails and most hand-rolled servers,
// and the names Flask and Uvicorn read instead.
func portEnv(port uint32) []string {
	p := strconv.Itoa(int(port))
	return []string{"PORT=" + p, "FLASK_RUN_PORT=" + p, "UVICORN_PORT=" + p}
}

var packageRunners = []string{"npm", "pnpm", "yarn", "bun"}

// scriptText is what the command really runs, for recognising the
// framework: the package.json script behind `npm run dev`, `pnpm dev`,
// `yarn dev` or `bun run dev`, else the command itself.
func scriptText(command []string, dir string) string {
	if script := scriptName(command); script != "" {
		var pkg struct {
			Scripts map[string]string `json:"scripts"`
		}
		if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil && json.Unmarshal(data, &pkg) == nil {
			if s, ok := pkg.Scripts[script]; ok {
				return s
			}
		}
	}
	return strings.Join(command, " ")
}

// scriptName is the package.json script a package-runner command runs,
// or "".
func scriptName(command []string) string {
	if len(command) < 2 || !slices.Contains(packageRunners, filepath.Base(command[0])) {
		return ""
	}
	switch command[1] {
	case "run", "run-script":
		if len(command) > 2 {
			return command[2]
		}
		return ""
	case "start", "dev", "serve":
		return command[1]
	}
	if filepath.Base(command[0]) != "npm" && !strings.HasPrefix(command[1], "-") {
		return command[1] // pnpm dev, yarn dev, bun dev
	}
	return ""
}

var (
	viteWords    = regexp.MustCompile(`(?:^|[\s/])(?:vite|vinxi|remix vite:dev|svelte-kit dev|react-router dev)(?:\s|$)`)
	astroWords   = regexp.MustCompile(`(?:^|[\s/])astro(?:\s+dev)?(?:\s|$)`)
	angularWords = regexp.MustCompile(`(?:^|[\s/])ng\s+serve(?:\s|$)`)
)

// runFramework is the framework a script starts, as far as its port
// flags go: "Vite", "Astro", "Angular", or "" for one that reads PORT.
func runFramework(script string) string {
	switch {
	case astroWords.MatchString(script):
		return "Astro"
	case angularWords.MatchString(script):
		return "Angular"
	case viteWords.MatchString(script):
		return "Vite"
	}
	return ""
}

// ownPort is the port a command names itself (--port 3000, -p 3000).
func ownPort(command []string) (uint32, bool) {
	for i, a := range command {
		v, ok := strings.CutPrefix(a, "--port=")
		if !ok && (a == "--port" || a == "-p") && i+1 < len(command) {
			v, ok = command[i+1], true
		}
		if ok {
			if n, err := strconv.ParseUint(v, 10, 16); err == nil && n > 0 {
				return uint32(n), true
			}
		}
	}
	return 0, false
}

// withPortFlags is the command with the flags its framework takes the
// port by — after the `--` npm needs to pass them on to the script —
// unless it already names a port.
func withPortFlags(command []string, framework string, port uint32) []string {
	if _, own := ownPort(command); own {
		return command
	}
	var flags []string
	p := strconv.Itoa(int(port))
	switch framework {
	case "Vite":
		flags = []string{"--port", p, "--strictPort"}
	case "Astro", "Angular":
		flags = []string{"--port", p}
	default:
		return command
	}
	out := slices.Clone(command)
	if filepath.Base(command[0]) == "npm" && scriptName(command) != "" && !slices.Contains(command, "--") {
		out = append(out, "--")
	}
	return append(out, flags...)
}

// forwardSignals passes a stop on to the command and everything it
// started. A package runner doesn't: SIGTERM to `npm run dev` ends npm
// and leaves Vite holding the port, the very leftover ports exists to
// find — so the tree is collected before anything is signalled, while
// the servers are still npm's children to find. Ctrl-C reaches them all
// from the terminal on its own, so SIGINT is passed only when there's no
// terminal to send it — an agent or script interrupting ports run.
func forwardSignals(p *os.Process) (stop func()) {
	sig := make(chan os.Signal, 1)
	signals := []os.Signal{os.Interrupt}
	if runtime.GOOS != "windows" {
		signals = append(signals, syscall.SIGTERM, syscall.SIGHUP)
	}
	signal.Notify(sig, signals...)
	tty := term.IsTerminal(int(os.Stdin.Fd()))
	done := make(chan struct{})
	go func() {
		for {
			select {
			case s := <-sig:
				if s == os.Interrupt && tty {
					continue
				}
				tree := descendants(int32(p.Pid))
				_ = p.Signal(s)
				for _, d := range tree {
					_ = d.SendSignal(s.(syscall.Signal))
				}
			case <-done:
				return
			}
		}
	}()
	return func() {
		signal.Stop(sig)
		close(done)
	}
}

// descendants is every process under pid, children first.
func descendants(pid int32) []*gproc.Process {
	root, err := gproc.NewProcess(pid)
	if err != nil {
		return nil
	}
	var out []*gproc.Process
	queue := []*gproc.Process{root}
	for len(queue) > 0 {
		kids, _ := queue[0].Children()
		queue = append(queue[1:], kids...)
		out = append(out, kids...)
	}
	return out
}
