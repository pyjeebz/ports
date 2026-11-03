package cmd

import (
	"fmt"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

// free is the port half of `ports stop`, kept for the scripts that have
// called it since v0.1.0 — hidden from help, its behavior unchanged.
var freeCmd = &cobra.Command{
	Use:    "free <port>",
	Short:  "Kill whatever holds a port (now `ports stop <port>`)",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := requirePort(args[0])
		if err != nil {
			return err
		}
		return freePort(port)
	},
}

// freePort stops whatever listens on port.
func freePort(port uint32) error {
	if err := stopFormat(); err != nil {
		return err
	}
	cfg := buildInspectConfig()
	services, err := inspect.Scan(cfg)
	if err != nil {
		return err
	}
	matched := match(services, func(s inspect.Service) bool { return s.Port == port })
	if len(matched) == 0 {
		return noneFound(fmt.Sprintf("nothing listening on %d — run `ports doctor %d` to see what holds it", port, port))
	}
	code := stopAll(matched, stopOptions(), func(s inspect.Service, _ string) string {
		if s.Container != "" {
			return fmt.Sprintf("✓ stopped %s (container %s) — port %d is free", inspect.ProcessName(s), inspect.DockerName(s), port)
		}
		return fmt.Sprintf("✓ killed %s (pid %d) — port %d is free", inspect.ProcessName(s), s.Pid, port)
	})
	if code != 0 {
		return exitError{code: code}
	}
	return nil
}

func init() {
	stopFlags(freeCmd)
	rootCmd.AddCommand(freeCmd)
}
