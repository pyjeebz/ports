package cmd

import (
	"fmt"
	"os"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

var killCmd = &cobra.Command{
	Use:   "kill <name>",
	Short: "Stop a project or service by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		q := args[0]
		services, err := inspect.Scan()
		if err != nil {
			return err
		}
		matched := match(services, func(s inspect.Service) bool { return s.MatchesName(q) })
		if len(matched) == 0 {
			fmt.Fprintf(os.Stderr, "ports: nothing matching %q — try `ports find %s`\n", q, q)
			os.Exit(1)
		}
		os.Exit(stopAll(matched, force, dryRun, func(s inspect.Service, ports string) string {
			return fmt.Sprintf("✓ stopped %s (%s)", process(s), ports)
		}))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
}