package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <port|name>",
	Short: "Open a running web server in the browser",
	Long: `ports open opens what a listener serves in your default browser:

  $ ports open 3000
  → opening http://localhost:3000/

A port (3000 or :3000) opens that port; a name opens the web server
` + "`ports stop`" + ` would match — a project, framework, container, or
process. ports asks the listener first, HTTPS then HTTP, so the browser
gets the scheme it really speaks — and a port that isn't a web server
(a database, a debugger) is reported, not opened.

A name with several web servers behind it opens a picker in a terminal;
off one, it's a usage error that lists the ports, so name one.

$BROWSER is honoured; otherwise it's open on macOS, xdg-open on Linux,
wslview or explorer.exe in WSL, and the URL handler on Windows. ports
opens a browser, never a server: it's an observer still.

Exit codes: 0 when the browser was asked to open the page, 1 when
nothing is listening, nothing answers HTTP, or no browser could be
started (the URL is printed either way), 2 for a bad invocation or a
name with several web servers off a terminal, 130 when the picker is
left.`,
	Args: cobra.ExactArgs(1),
	RunE: runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)
}

// openURL is the seam the browser launch runs through — tests swap it.
var openURL = launchBrowser

// webRow is a listener that answers HTTP, and the URL it answered on.
type webRow struct {
	inspect.Service
	url string
}

func runOpen(cmd *cobra.Command, args []string) error {
	port, isPort, err := portArg(args[0])
	if err != nil {
		return err
	}
	services, err := inspect.Scan(buildInspectConfig())
	if err != nil {
		return err
	}
	rows := match(services, func(s inspect.Service) bool {
		if isPort {
			return s.Port == port
		}
		return s.MatchesName(args[0])
	})
	if len(rows) == 0 {
		if isPort && !inspect.Listening(port) {
			return exitError{code: 1, msg: fmt.Sprintf("nothing listening on :%d", port)}
		}
		if isPort {
			// listening, not yet attributable to a process: ask it anyway
			rows = []inspect.Service{{Port: port}}
		} else {
			return exitError{code: 1, msg: fmt.Sprintf("nothing matching %q is listening — `ports` lists what is", args[0])}
		}
	}

	web := probeWeb(rows)
	switch {
	case len(web) == 0 && isPort:
		return exitError{code: 1, msg: fmt.Sprintf("%s (:%d) doesn't answer HTTP — it isn't a web server, or isn't serving yet; `ports doctor %d` looks closer",
			inspect.ProcessName(rows[0]), port, port)}
	case len(web) == 0:
		return exitError{code: 1, msg: fmt.Sprintf("%q holds %s — none answers HTTP", args[0], portList(distinctPorts(rows)))}
	case len(web) == 1:
		return open(web[0].url)
	}

	var ports []uint32
	for _, w := range web {
		ports = append(ports, w.Port)
	}
	if !interactive() {
		return exitError{code: 2, msg: fmt.Sprintf("%q answers HTTP on %s — name the port, e.g. `ports open %d`", args[0], portList(ports), ports[0])}
	}
	candidates := make([]inspect.Service, len(web))
	for i, w := range web {
		candidates[i] = w.Service
	}
	s, err := pickOnTerminal(candidates, pickSpec{verb: "open"})
	if errors.Is(err, errPickCancelled) {
		return exitError{code: 130, msg: "cancelled — nothing was opened"}
	}
	if err != nil {
		return exitError{code: 1, msg: err.Error()}
	}
	fmt.Println()
	return open(web[slices.IndexFunc(web, func(w webRow) bool { return w.Port == s.Port })].url)
}

// probeWeb asks each distinct port, all at once, whether it answers HTTP
// — one row per port, in port order.
func probeWeb(rows []inspect.Service) []webRow {
	seen := map[uint32]bool{}
	var unique []inspect.Service
	for _, s := range rows {
		if !seen[s.Port] {
			seen[s.Port] = true
			unique = append(unique, s)
		}
	}
	slices.SortStableFunc(unique, func(a, b inspect.Service) int { return int(a.Port) - int(b.Port) })
	urls := make([]string, len(unique))
	var wg sync.WaitGroup
	for i, s := range unique {
		wg.Add(1)
		go func() {
			defer wg.Done()
			urls[i], _ = inspect.WebURL(s.Port, s.Addr)
		}()
	}
	wg.Wait()
	var web []webRow
	for i, s := range unique {
		if urls[i] != "" {
			web = append(web, webRow{s, urls[i]})
		}
	}
	return web
}

func distinctPorts(rows []inspect.Service) []uint32 {
	var ports []uint32
	for _, s := range rows {
		if !slices.Contains(ports, s.Port) {
			ports = append(ports, s.Port)
		}
	}
	slices.Sort(ports)
	return ports
}

func open(url string) error {
	fmt.Printf("→ opening %s\n", url)
	if err := openURL(url); err != nil {
		return exitError{code: 1, msg: fmt.Sprintf("couldn't start a browser: %v — the page is at %s", err, url)}
	}
	return nil
}

// browserCommand is how this machine opens a URL: $BROWSER first (its
// first entry, with %s for the URL or the URL appended), else the
// platform's own opener.
func browserCommand(url string) (*exec.Cmd, error) {
	if b := strings.TrimSpace(strings.Split(os.Getenv("BROWSER"), string(os.PathListSeparator))[0]); b != "" {
		parts := strings.Fields(b)
		if strings.Contains(b, "%s") {
			for i := range parts {
				parts[i] = strings.ReplaceAll(parts[i], "%s", url)
			}
		} else {
			parts = append(parts, url)
		}
		return exec.Command(parts[0], parts[1:]...), nil
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url), nil
	case "windows":
		// not `start`: cmd.exe would read & in a URL as a command separator
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url), nil
	}
	if inspect.OnWSL() {
		if _, err := exec.LookPath("wslview"); err == nil {
			return exec.Command("wslview", url), nil
		}
		return exec.Command("explorer.exe", url), nil
	}
	if _, err := exec.LookPath("xdg-open"); err != nil {
		return nil, errors.New("no xdg-open here, and $BROWSER is unset")
	}
	return exec.Command("xdg-open", url), nil
}

// launchBrowser starts the opener and lets it go. Openers usually hand
// off and exit at once; one that runs the browser in the foreground is
// left running. A quick failure — a bad $BROWSER, xdg-open with no
// desktop — is reported.
func launchBrowser(url string) error {
	c, err := browserCommand(url)
	if err != nil {
		return err
	}
	if err := c.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("%s: %w", c.Args[0], err)
		}
		return nil
	case <-time.After(2 * time.Second):
		return nil // still running: the browser itself, in the foreground
	}
}
