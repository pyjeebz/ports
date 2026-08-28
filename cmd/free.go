package cmd

import (
	"fmt"
	"os"
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
			fmt.Fprintf(os.Stderr, "ports: invalid port %q\n", arg)
			os.Exit(2)
		}
		cfg := buildInspectConfig()
		services, err := inspect.Scan(cfg)
		if err != nil {
			return err
		}
		matched := match(services, func(s inspect.Service) bool { return s.Port == uint32(port) })
		if len(matched) == 0 {
			fmt.Fprintf(os.Stderr, "ports: nothing listening on %d\n", port)
			os.Exit(1)
		}
		os.Exit(stopAll(matched, force, dryRun, func(s inspect.Service, _ string) string {
			if s.Container != "" {
				return fmt.Sprintf("✓ stopped %s (container %s) — port %d is free", process(s), dockerName(s), port)
			}
			return fmt.Sprintf("✓ killed %s (pid %d) — port %d is free", process(s), s.Pid, port)
		}))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(freeCmd)
}