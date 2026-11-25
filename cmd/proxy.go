package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/pyjeebz/ports/internal/naming"
	"github.com/pyjeebz/ports/internal/router"
	"github.com/pyjeebz/ports/internal/routes"
	"github.com/spf13/cobra"
)

var proxyPort int

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Reach every dev server by name — http://shop.localhost",
	Long: `ports proxy runs the router in the foreground: every web dev server on
this machine answers at http://<name>.localhost, whether ports started it
or not, until Ctrl-C.

  $ ports proxy
  routing *.localhost on :80 — Ctrl-C stops it

    shop.localhost        :3000   Next.js   ~/shop
    api.shop.localhost    :8000   FastAPI   ~/shop

A name comes from the project — its package.json name or its folder —
and the app's place in it: the main app is shop, an app in apps/api is
api.shop, the same app in a git worktree on fix-ui is fix-ui.shop. Names
don't come from ports, so a server that restarts on another port keeps
its name; the router finds where it went within two seconds.

The router listens on 127.0.0.1 and ::1 only, and answers only
.localhost names it knows. It prefers port 80, so URLs need no port;
when 80 is taken or not allowed it uses 8480 instead (--port picks one).
It starts nothing: servers run the way they always have.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProxy(cmd.Flags().Changed("port"))
	},
}

func init() {
	proxyCmd.Flags().IntVar(&proxyPort, "port", 0, "listen on this port instead of 80 (no fallback)")
	rootCmd.AddCommand(proxyCmd)
}

func runProxy(portSet bool) error {
	if portSet && (proxyPort < 1 || proxyPort > 65535) {
		return exitError{code: 2, msg: fmt.Sprintf("--port %d is out of range (1-65535)", proxyPort)}
	}
	if runtime.GOOS != "windows" && os.Geteuid() == 0 {
		// as root the scan sees every user's processes, and the router
		// would route to servers that aren't yours
		return exitError{code: 2, msg: "ports proxy runs as you, not root — it routes only your own dev servers"}
	}
	if info, ok := router.Running(); ok {
		return exitError{code: 1, msg: fmt.Sprintf("a router already runs on :%d (pid %d)", info.Port, info.Pid)}
	}

	var (
		listeners []net.Listener
		port      int
		why, err  error
	)
	if portSet {
		port = proxyPort
		listeners, err = router.Listen(port)
	} else {
		listeners, port, why, err = router.ListenDefault()
	}
	if err != nil {
		return fmt.Errorf("can't listen: %w", err)
	}

	cfg := buildInspectConfig()
	table := routes.NewTable(func() ([]inspect.Service, error) { return inspect.ScanOwn(cfg) }, naming.LoadClaims(), routes.LoadDeclared)
	if _, err := table.Refresh(); err != nil {
		return err
	}
	if err := router.Announce(port); err != nil {
		fmt.Fprintln(os.Stderr, "ports: can't record the router:", err)
	}
	defer router.Withdraw()

	srv := &http.Server{Handler: router.New(table, port), ReadHeaderTimeout: 10 * time.Second}
	serveErr := make(chan error, len(listeners))
	for _, l := range listeners {
		go func() { serveErr <- srv.Serve(l) }()
	}

	fmt.Println(inspect.Mark(fmt.Sprintf("✓ routing *.localhost on :%d — Ctrl-C stops it", port), false))
	if why != nil {
		fmt.Fprintf(os.Stderr, "  port 80 isn't available (%s), so names carry :%d\n", bindReason(why), port)
		if fix := port80Fix(why); fix != "" {
			fmt.Fprintln(os.Stderr, "  for URLs with no port:", fix)
		}
	}
	printRoutes(table.All(), port)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		table.Run(2*time.Second, stop, func(changes []routes.Change) { printChanges(changes, port) })
	})

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)
	select {
	case <-sig:
	case err = <-serveErr:
	}
	close(stop)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	wg.Wait()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func printRoutes(rs []routes.Route, port int) {
	fmt.Println()
	if len(rs) == 0 {
		fmt.Println("  no dev servers yet — start one the way you always do, and it shows up here")
		fmt.Println()
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	for _, r := range rs {
		_, _ = fmt.Fprintf(w, "  %s\t:%d\t%s\t%s\n", router.URL(r.Name, port), r.Port, r.Framework, inspect.ShortPath(r.Project))
	}
	_ = w.Flush()
	fmt.Println()
}

func printChanges(changes []routes.Change, port int) {
	for _, c := range changes {
		url := router.URL(c.Route.Name, port)
		switch c.Kind {
		case "added":
			fmt.Printf("  + %s → :%d %s\n", url, c.Route.Port, c.Route.Framework)
		case "moved":
			fmt.Printf("  ~ %s → :%d %s\n", url, c.Route.Port, c.Route.Framework)
		case "removed":
			fmt.Printf("  - %s\n", url)
		}
	}
}

// bindReason is why port 80 refused, in a few words.
func bindReason(err error) string {
	switch {
	case errors.Is(err, os.ErrPermission):
		return "not allowed without privileges"
	case strings.Contains(err.Error(), "address already in use"), strings.Contains(err.Error(), "Only one usage"):
		return "something else holds it — `ports 80` shows what"
	default:
		return err.Error()
	}
}

// port80Fix is the one-time step that lets the router have port 80, when
// the reason it couldn't is privileges — Linux reserves ports below 1024.
func port80Fix(err error) string {
	if runtime.GOOS != "linux" || !errors.Is(err, os.ErrPermission) {
		return ""
	}
	exe, e := os.Executable()
	if e != nil {
		exe = "$(command -v ports)"
	}
	return "sudo setcap cap_net_bind_service=+ep " + exe
}
