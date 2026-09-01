package cmd

import (
	"fmt"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff <snapshot>",
	Short: "Compare the machine now against a saved snapshot",
	Long: `ports diff answers "what changed since I looked?" — the snapshot is a
moment of the machine you saved, and diff reads the same table against
that moment instead of nothing.

  $ ports --format snapshot > before.json
  $ ports diff before.json
  PORT    CHANGE   PROCESS     PROJECT          NOTE
  :3000   new      Next.js     ~/projects/shop  listening now — not in the snapshot
  :5432   gone     PostgreSQL  docker: shop-db  in the snapshot, no longer listening
  :8000   changed  FastAPI     ~/projects/shop  restarted — pid 48391 → 51234

The same filters scope both sides of the comparison, so ` + "`ports diff before.json --port 3000-9000`" + `
asks about one range of the machine — rows outside the range are left
out of the comparison, not reported gone.

Exit codes: 0 when the machine matches the snapshot, 1 when anything
changed (grep-style, so ` + "`ports diff before.json || ./investigate.sh`" + `
gates naturally), and 2 for a bad invocation — a missing snapshot, one
that doesn't parse, or one from a newer ports.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if watch {
			return errWatchNotList
		}
		// validate the arguments before paying for a scan
		filter, err := inspect.ParseFilter(filterPorts, filterFrameworks, filterProjects, filterUptime)
		if err != nil {
			return exitError{code: 2, msg: err.Error()}
		}
		snap, err := inspect.LoadSnapshot(args[0])
		if err != nil {
			return exitError{code: 2, msg: err.Error()}
		}
		// the filter scopes the whole comparison, not just the now-scan —
		// a snapshot row outside the range is left out, never reported gone
		was := inspect.ApplyFilter(snap.Services, filter)
		cfg := buildInspectConfig()
		now, err := inspect.Scan(cfg)
		if err != nil {
			return err
		}
		diffs := inspect.DiffServices(was, inspect.ApplyFilter(now, filter))

		switch {
		case jsonOut || format == "json":
			if err := inspect.WriteDiffsJSON(diffs); err != nil {
				return err
			}
		case format == "table":
			inspect.WriteDiffs(diffs, was, snap.GeneratedAt)
		default:
			return exitError{code: 2, msg: fmt.Sprintf("--format %q — diff renders table or json", format)}
		}
		if len(diffs) > 0 {
			return exitError{code: 1}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
}