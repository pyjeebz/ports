package cmd

import (
	"errors"
	"fmt"
	"slices"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <port|name>",
	Short: "Stop what holds a port, or everything a name matches",
	Long: `ports stop ends what you name. A port (3000 or :3000) stops whatever
listens on it; anything else is a name — a project, framework, container,
systemd unit, or process — and stops every service it matches. Anything
all-digits is read as a port: a project named 2048 stops by name with
` + "`ports kill 2048`" + `.

  $ ports stop 3000
  ✓ killed Next.js (pid 48391) — port 3000 is free

  $ ports stop shop
  ✓ stopped Next.js (:3000)
  ✓ stopped FastAPI (:8000)

SIGTERM first, SIGKILL only if it doesn't exit; --force skips straight
to SIGKILL. Containers stop through the engine, never docker-proxy.
Another user's listener can't be stopped without sudo: a name that
reaches one skips it and says which command would.

--expect <name> stops only if everything it would stop is that name —
the guard for scripts and agents. --dry-run shows what would stop.

Run bare in a terminal, ports stop opens a picker: choose a row with
↑/↓, confirm, and it stops exactly the process you confirmed — if it
changed while you were choosing, nothing is stopped. Scripts and pipes
never see it — there, bare ports stop is a usage error.

Exits 0 when the stop did its job, 1 when nothing matched, a stop
failed, or --expect refused it, 2 for a bad invocation, 130 when the
picker is left without choosing.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return pickToStop()
		}
		port, isPort, err := portArg(args[0])
		if err != nil {
			return err
		}
		if isPort {
			return freePort(port)
		}
		return killName(args[0])
	},
}

func init() {
	stopFlags(stopCmd)
	rootCmd.AddCommand(stopCmd)
}

// pickToStop is bare `ports stop`: the picker in a terminal, a usage
// error anywhere else — a script that forgot its argument must fail
// loudly, never wait on a prompt.
func pickToStop() error {
	if !interactive() || wantJSON() {
		return exitError{code: 2, msg: "`ports stop` needs a port or a name — e.g. `ports stop 3000`"}
	}
	if err := stopFormat(); err != nil {
		return err
	}
	services, err := inspect.Scan(buildInspectConfig())
	if err != nil {
		return err
	}
	if len(services) == 0 {
		return noneFound("nothing is listening — nothing to stop")
	}
	slices.SortStableFunc(services, func(a, b inspect.Service) int { return int(a.Port) - int(b.Port) })
	s, err := pickAndConfirm(services)
	if errors.Is(err, errPickCancelled) {
		return exitError{code: 130, msg: err.Error()}
	}
	if err != nil {
		return exitError{code: 1, msg: err.Error()}
	}
	fmt.Println()
	return stopPicked(s)
}

// rescan is the scan stopPicked trusts — a seam tests swap to change the
// machine between the pick and the stop.
var rescan = func() ([]inspect.Service, error) { return inspect.Scan(buildInspectConfig()) }

// stopPicked stops exactly what was confirmed. The machine moved on while
// the picker was open — a dev server restarted, another project took the
// port — so it scans again and stops only the confirmed target (its
// container, its pid and start time, or its unit), never whatever holds
// the port now; a target no longer holding the port stops nothing.
func stopPicked(chosen inspect.Service) error {
	if err := stopFormat(); err != nil {
		return err
	}
	services, err := rescan()
	if err != nil {
		return err
	}
	matched := match(services, func(s inspect.Service) bool { return sameTarget(chosen, s) })
	if !slices.ContainsFunc(matched, func(s inspect.Service) bool { return s.Port == chosen.Port }) {
		return noneFound(fmt.Sprintf("%s (:%d) changed while you were choosing — nothing was stopped; run `ports stop` again",
			inspect.ProcessName(chosen), chosen.Port))
	}
	code := stopAll(matched, stopOptions(), func(s inspect.Service, ports string) string {
		return fmt.Sprintf("✓ stopped %s (%s)", inspect.ProcessName(s), ports)
	})
	if code != 0 {
		return exitError{code: code}
	}
	return nil
}
