package cmd

import (
	"fmt"
	"os"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

var findCmd = &cobra.Command{
	Use:   "find <name>",
	Short: "Filter by framework, process, or project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		q := args[0]
		cfg := buildInspectConfig()
		services, err := inspect.Scan(cfg)
		if err != nil {
			return err
		}
		matched := match(services, func(s inspect.Service) bool { return s.MatchesQuery(q) })
		if len(matched) == 0 {
			fmt.Fprintf(os.Stderr, "ports: nothing matching %q\n", q)
			os.Exit(1)
		}
		if jsonOut {
			return inspect.WriteJSON(matched)
		}
		inspect.WriteTable(matched)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(findCmd)
}