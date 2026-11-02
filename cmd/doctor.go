package cmd

import (
	"fmt"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor [port]",
	Short: "Diagnose a port — or the whole machine — and say what to do next",
	Long: `ports doctor explains what a port's row means, and why a stop failed
when one did. It never acts — every finding is an observation, every fix
is a command for you to run.

  $ ports doctor 3000
  :3000  Next.js  pid 48391  ~/projects/shop  up 14m
  ✓ listening on :3000
  ✓ process state R — signals land
  ✓ pid 48391 still matches the scan — no pid reuse
  ✓ nothing else holds :3000 — no stray sockets

Without a port, it checks the machine: the container engine, every
docker-proxy port that didn't resolve, and every listener a plain scan
can't see or shouldn't share.

Exit codes: 0 when nothing needs doing (every finding is ✓ or · info),
1 when anything is ⚠ or ✗
(grep-style, so ` + "`ports doctor 3000 && ./deploy.sh`" + ` gates
naturally), 2 for bad arguments.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := buildInspectConfig()

		var findings []inspect.Finding
		var render func()
		switch len(args) {
		case 0:
			var err error
			if findings, err = inspect.DiagnoseMachine(cfg); err != nil {
				return err
			}
			render = func() { inspect.WriteMachineFindings(findings) }
		default:
			port, err := requirePort(args[0])
			if err != nil {
				return err
			}
			rows, f, err := inspect.Diagnose(cfg, port)
			if err != nil {
				return err
			}
			findings = f
			render = func() { inspect.WriteDiagnosis(rows, findings) }
		}

		// doctor's findings have no csv/tsv/snapshot render — a format
		// it can't draw is a bad invocation, not a silent table
		switch format {
		case "table", "json":
		default:
			return exitError{code: 2, msg: fmt.Sprintf("--format %q — doctor renders table or json", format)}
		}
		if jsonOut || format == "json" {
			if err := inspect.WriteFindingsJSON(findings); err != nil {
				return err
			}
		} else {
			render()
		}
		if inspect.HasProblems(findings) {
			return exitError{code: 1}
		}
		return nil
	},
}

func init() {
	formatFlags(doctorCmd, "table, json")
	rootCmd.AddCommand(doctorCmd)
}
