package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/pyjeebz/ports/internal/inspect"
)

// minWatchInterval keeps the loop from spinning: below this a rescan
// costs more than the pause between rescans.
const minWatchInterval = 250 * time.Millisecond

// errWatchNotList answers --watch on a subcommand where it would silently
// no-op: it only means something for listing.
var errWatchNotList = exitError{code: 2, msg: "--watch only applies to listing (ports, ports list)"}

// stdoutIsTTY reports whether stdout is an interactive terminal. Watch's
// table mode clears the screen, which is noise in a pipe.
func stdoutIsTTY() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && (fi.Mode()&os.ModeCharDevice) != 0
}

// validateWatch rejects the combinations that can't loop: intervals fast
// enough to stack scans, formats that have no redraw story, and a table
// render with nowhere to draw it.
func validateWatch(interval time.Duration, format string, jsonOut, tty bool) error {
	if interval < minWatchInterval {
		return exitError{code: 2, msg: fmt.Sprintf("--interval must be at least %s, got %s", minWatchInterval, interval)}
	}
	if jsonOut {
		format = "json" // --json is deprecated but still honored
	}
	if format != "table" && format != "json" {
		return exitError{code: 2, msg: fmt.Sprintf("--watch needs table or json output, got format %q", format)}
	}
	// --tree is documented as table-format-only and ignored elsewhere,
	// same as a one-shot run
	if format == "table" && !tty {
		return exitError{code: 2, msg: "--watch needs a terminal — use --format json for piped output"}
	}
	return nil
}

// watchLoop runs tick, waits out the interval, and repeats — scan-then-sleep,
// never a ticker: a scan can legitimately take longer than the interval
// (the WSL interop probe alone allows 3s), and stacking scans would only
// grow the backlog. It returns nil when ctx is done or the tick fails.
func watchLoop(ctx context.Context, interval time.Duration, tick func() error) error {
	for {
		if err := tick(); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(interval):
		}
	}
}

// runWatch is the --watch path of listing: validate the combination once,
// then loop the scan+render tick until interrupted. The first SIGINT
// cancels gracefully — the in-flight scan finishes, then the loop exits;
// a second SIGINT exits immediately.
func runWatch() error {
	if err := validateWatch(watchInterval, format, jsonOut, stdoutIsTTY()); err != nil {
		return err
	}
	// sorted once, honored on every redraw — the order a watch opens in
	// is the order it keeps
	sort, err := resolveSort()
	if err != nil {
		return err
	}

	// Parse filters once, before the first scan — a bad filter should
	// fail before any output, not mid-loop.
	filter, err := inspect.ParseFilter(filterPorts, filterFrameworks, filterProjects, filterUptime)
	if err != nil {
		return exitError{code: 2, msg: err.Error()}
	}
	cfg := buildInspectConfig()
	asJSON := jsonOut || format == "json"

	tick := func() error {
		services, err := inspect.Scan(cfg)
		if err != nil {
			return err
		}
		services = inspect.ApplyFilter(services, filter)
		if asJSON {
			// one array per tick, newline-delimited — no clears
			return inspect.WriteJSON(services)
		}
		// clear the screen, then redraw — identical bytes to a one-shot run
		fmt.Print("\033[2J\033[H")
		if noMatch(services, filter) {
			fmt.Println(noMatchMsg)
			return nil
		}
		if showTree {
			inspect.WriteTree(services)
			return nil
		}
		inspect.WriteTable(services, sort)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigch := make(chan os.Signal, 2)
	signal.Notify(sigch, os.Interrupt)
	defer signal.Stop(sigch)
	go func() {
		<-sigch // first: graceful — the current scan finishes, then we exit
		cancel()
		<-sigch // second: the scan is taking too long; don't wait for it
		os.Exit(130)
	}()

	return watchLoop(ctx, watchInterval, tick)
}