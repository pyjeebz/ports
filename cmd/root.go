package cmd

import (
	"fmt"
	"os"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

var (
	jsonOut bool
	force   bool
	dryRun  bool
)

var rootCmd = &cobra.Command{
	Use:   "ports",
	Short: "See what's running on your machine, by project — and kill it by name",
	Long: `ports answers one question, instantly and with zero configuration:
what is running on this machine right now, and what does it belong to?

  $ ports
  PORT   PROCESS      PROJECT              UPTIME
  3000   Next.js      ~/projects/shop      14m
  5432   PostgreSQL   docker: shop-db      2d
  8000   FastAPI      ~/projects/shop      14m
  5173   Vite         ~/projects/admin     3h

  $ ports kill shop
  ✓ stopped Next.js (:3000)
  ✓ stopped FastAPI (:8000)

  $ ports free 3000
  ✓ killed Next.js (pid 48391) — port 3000 is free`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd, args)
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "ports:", err)
		os.Exit(1)
	}
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ps"},
	Short:   "List all listening ports with framework, project, and uptime",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd, args)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "JSON output")
	rootCmd.PersistentFlags().BoolVar(&force, "force", false, "SIGKILL immediately instead of SIGTERM first")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would be killed without doing it")
	rootCmd.Flags().BoolP("version", "v", false, "Print version")
	rootCmd.SetVersionTemplate("ports {{.Version}}\n")

	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	services, err := inspect.Scan()
	if err != nil {
		return err
	}

	if jsonOut {
		return inspect.WriteJSON(services)
	}
	inspect.WriteTable(services)
	return nil
}