package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

var (
	waitTimeout time.Duration
	waitGone    bool
	waitHTTP    bool
)

// Poll cadences. A port is a read of the kernel's socket tables, cheap
// enough to ask four times a second; a name needs a full scan for its
// project context, so it is asked less often — and never stacked, the
// next poll starts only after the last one finished.
const (
	waitPortPoll = 250 * time.Millisecond
	waitNamePoll = 500 * time.Millisecond
)

var waitCmd = &cobra.Command{
	Use:   "wait <port|name>",
	Short: "Block until a port or service is listening — or, with --gone, free",
	Long: `ports wait blocks until something listens, then prints its row — the
step between starting a server and using it, without a sleep-and-retry.

  $ npm run dev &
  $ ports wait 3000 --http
  PORT   PROCESS   PROJECT   UPTIME
  3000   Next.js   shop      3s

A port (3000 or :3000) waits for any listener on it; a name waits for
a service that ` + "`ports stop`" + ` would match — a project, framework,
container, or process. --http waits until the port answers an HTTP request, not just
accepts connections: dev servers often bind before they can serve.
--gone waits the other way, until the port or service is free.

ports wait only watches. It never starts anything.

Exit codes: 0 when the condition is met, 1 when --timeout passes first
(grep-style, so ` + "`ports wait 3000 || exit 1`" + ` gates a script),
2 for a bad invocation, 130 when interrupted.`,
	Args: cobra.ExactArgs(1),
	RunE: runWait,
}

func init() {
	formatFlags(waitCmd, "table, json")
	waitCmd.Flags().DurationVar(&waitTimeout, "timeout", 30*time.Second, "Give up after this long (0 waits forever)")
	waitCmd.Flags().BoolVar(&waitGone, "gone", false, "Wait until nothing listens instead")
	waitCmd.Flags().BoolVar(&waitHTTP, "http", false, "Wait until the port answers HTTP, not just connections (ports only)")
	rootCmd.AddCommand(waitCmd)
}

// waitTarget is what wait watches: a port, or a name the way kill
// matches one.
type waitTarget struct {
	port uint32
	name string
}

func (t waitTarget) String() string {
	if t.name != "" {
		return fmt.Sprintf("%q", t.name)
	}
	return fmt.Sprintf(":%d", t.port)
}

// rows scans and keeps the target's rows.
func (t waitTarget) rows() ([]inspect.Service, error) {
	services, err := inspect.Scan(buildInspectConfig())
	if err != nil {
		return nil, err
	}
	return match(services, func(s inspect.Service) bool {
		if t.name != "" {
			return s.MatchesName(t.name)
		}
		return s.Port == t.port
	}), nil
}

func runWait(cmd *cobra.Command, args []string) error {
	if format != "table" && format != "json" {
		return exitError{code: 2, msg: fmt.Sprintf("--format %q — wait renders table or json", format)}
	}
	if waitTimeout < 0 {
		return exitError{code: 2, msg: fmt.Sprintf("--timeout must not be negative, got %s", waitTimeout)}
	}
	var t waitTarget
	port, isPort, err := portArg(args[0])
	if err != nil {
		return err
	}
	if isPort {
		t.port = port
	} else {
		t.name = args[0]
	}
	if waitHTTP && t.name != "" {
		return exitError{code: 2, msg: "--http needs a port — a name can hold several"}
	}
	if waitHTTP && waitGone {
		return exitError{code: 2, msg: "--http and --gone don't combine — a free port answers nothing"}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if waitTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, waitTimeout)
		defer cancel()
	}
	interrupted := make(chan struct{})
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	defer signal.Stop(sigch)
	go func() {
		select {
		case <-sigch:
			close(interrupted)
			cancel()
		case <-ctx.Done():
		}
	}()

	poll := waitPortPoll
	if t.name != "" {
		poll = waitNamePoll
	}
	for {
		rows, done, err := t.check()
		if err != nil {
			return err
		}
		if done {
			return waitReport(t, rows)
		}
		select {
		case <-interrupted:
			return exitError{code: 130}
		case <-ctx.Done():
			select {
			case <-interrupted:
				return exitError{code: 130}
			default:
			}
			return exitError{code: 1, msg: t.timeoutMsg()}
		case <-time.After(poll):
		}
	}
}

// check asks once whether the wait is over. A port answers from the
// cheap socket read; only once it is listening does a scan describe it.
func (t waitTarget) check() ([]inspect.Service, bool, error) {
	if t.name == "" {
		listening := inspect.Listening(t.port)
		if waitGone {
			return nil, !listening, nil
		}
		if !listening || (waitHTTP && !inspect.ServesHTTP(t.port)) {
			return nil, false, nil
		}
	}
	rows, err := t.rows()
	if err != nil {
		return nil, false, err
	}
	if waitGone {
		return nil, len(rows) == 0, nil
	}
	// a port can be listening and still missing from the scan for a
	// moment (bound, not yet attributable) — keep waiting for the row
	return rows, len(rows) > 0, nil
}

func waitReport(t waitTarget, rows []inspect.Service) error {
	if waitGone {
		if wantJSON() {
			return inspect.WriteJSON(nil)
		}
		fmt.Printf("✓ %s is free\n", t)
		return nil
	}
	return writeReport(rows, inspect.SortDefault)
}

// timeoutMsg says what the machine looked like when the wait gave up,
// and the command that moves it along.
func (t waitTarget) timeoutMsg() string {
	after := fmt.Sprintf("timed out after %s", waitTimeout)
	if waitGone {
		rows, _ := t.rows()
		if len(rows) > 0 {
			s := rows[0]
			stop := fmt.Sprintf("ports stop %d", s.Port)
			if t.name != "" {
				stop = "ports stop " + t.name
			}
			return fmt.Sprintf("%s — %s still holds %s; `%s` stops it", after, inspect.ProcessName(s), t, stop)
		}
		return after + " — " + t.String() + " was still held"
	}
	if waitHTTP && inspect.Listening(t.port) {
		return fmt.Sprintf("%s — :%d is listening but not answering HTTP; `ports doctor %d` looks closer", after, t.port, t.port)
	}
	if t.name != "" {
		return fmt.Sprintf("%s — nothing matching %s is listening", after, t)
	}
	return fmt.Sprintf("%s — nothing listening on %s", after, t)
}
