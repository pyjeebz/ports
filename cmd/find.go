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
		filter, err := inspect.ParseFilter(filterPorts, filterFrameworks, filterProjects, filterUptime)
		if err != nil {
			return exitError{code: 2, msg: err.Error()}
		}
		cfg := buildInspectConfig()
		services, err := inspect.Scan(cfg)
		if err != nil {
			return err
		}
		services = inspect.ApplyFilter(services, filter)
		matched := match(services, func(s inspect.Service) bool { return s.MatchesQuery(q) })
		if len(matched) == 0 {
			return noneFound(fmt.Sprintf("nothing matching %q", q))
		}
		return writeReport(matched, sort)
	},
}

func init() {
	formatFlags(findCmd, allFormats)
	tableFlags(findCmd)
	filterFlags(findCmd)
	rootCmd.AddCommand(findCmd)
}
