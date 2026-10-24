package cmd

import (
	"fmt"
	"strconv"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

var freeCmd = &cobra.Command{
	Use:   "free <port>",
	Short: "Kill whatever holds a port",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := args[0]
		port, err := strconv.ParseUint(arg, 10, 16)
		if err != nil {
			return exitError{code: 2, msg: fmt.Sprintf("invalid port %q", arg)}
		}
		cfg := buildInspectConfig()
		services, err := inspect.Scan(cfg)
		if err != nil {
			return err
		}
		matched := match(services, func(s inspect.Service) bool { return s.Port == uint32(port) })
		if len(matched) == 0 {
			return fmt.Errorf("nothing listening on %d — run `ports doctor %d` to see what holds it", port, port)
		}
		code := stopAll(matched, force, dryRun, func(s inspect.Service, _ string) string {
			if s.Container != "" {
				return fmt.Sprintf("✓ stopped %s (container %s) — port %d is free", inspect.ProcessName(s), inspect.DockerName(s), port)
			}
			return fmt.Sprintf("✓ killed %s (pid %d) — port %d is free", inspect.ProcessName(s), s.Pid, port)
		})
		if code != 0 {
			return exitError{code: code}
		}
		return nil
	},
}

func init() {
	stopFlags(freeCmd)
	rootCmd.AddCommand(freeCmd)
}
