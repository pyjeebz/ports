package cmd

import (
	"fmt"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

var findCmd = &cobra.Command{
	Use:   "find <name>",
	Short: "Filter by framework, process, or project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		q := args[0]
		// validate the arguments before paying for a scan
		sort, err := resolveSort()
		if err != nil {
			return err
		}
		cfg := buildInspectConfig()
		services, err := inspect.Scan(cfg)
		if err != nil {
			return err
		}
		matched := match(services, func(s inspect.Service) bool { return s.MatchesQuery(q) })
		if len(matched) == 0 {
			return fmt.Errorf("nothing matching %q", q)
		}
		return writeReport(matched, sort)
	},
}

func init() {
	rootCmd.AddCommand(findCmd)
}
