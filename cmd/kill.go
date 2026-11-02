package cmd

import (
	"fmt"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

var killCmd = &cobra.Command{
	Use:   "kill <name>",
	Short: "Stop a project or service by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		q := args[0]
		if err := stopFormat(); err != nil {
			return err
		}
		cfg := buildInspectConfig()
		services, err := inspect.Scan(cfg)
		if err != nil {
			return err
		}
		matched := match(services, func(s inspect.Service) bool { return s.MatchesName(q) })
		if len(matched) == 0 {
			return noneFound(fmt.Sprintf("nothing matching %q — try `ports find %s`", q, q))
		}
		code := stopAll(matched, stopOptions(), func(s inspect.Service, ports string) string {
			return fmt.Sprintf("✓ stopped %s (%s)", inspect.ProcessName(s), ports)
		})
		if code != 0 {
			return exitError{code: code}
		}
		return nil
	},
}

func init() {
	stopFlags(killCmd)
	rootCmd.AddCommand(killCmd)
}
