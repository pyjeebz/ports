package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	jsonOut          bool
	force            bool
	dryRun           bool
	format           string
	showTree         bool
	filterPorts      string
	filterFrameworks string
	filterProjects   string
	filterUptime     string
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

	// Load config file
	if err := loadConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "ports: config:", err)
		os.Exit(1)
	}

	// Bind flags to viper so they can be set via config file
	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		fmt.Fprintln(os.Stderr, "ports: bind flags:", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		var ee exitError
		if errors.As(err, &ee) {
			// output already printed; just carry the code
			if ee.msg != "" {
				fmt.Fprintln(os.Stderr, "ports:", ee.msg)
			}
			os.Exit(ee.code)
		}
		fmt.Fprintln(os.Stderr, "ports:", err)
		os.Exit(1)
	}
}

func loadConfig() error {
	viper.SetConfigName("ports-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/ports")
	viper.AddConfigPath("/etc/ports")

	// Environment variable prefix
	viper.SetEnvPrefix("PORTS")
	viper.AutomaticEnv()

	// If config file found, read it
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
		// Config file not found is OK
	}
	return nil
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
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "JSON output (deprecated: use --format json)")
	rootCmd.PersistentFlags().BoolVar(&force, "force", false, "SIGKILL immediately instead of SIGTERM first")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would be killed without doing it")
	rootCmd.PersistentFlags().StringVar(&format, "format", "table", "Output format: table, json, csv, tsv")
	rootCmd.PersistentFlags().BoolVar(&showTree, "tree", false, "Nest child processes under their parent (table format only)")
	rootCmd.PersistentFlags().StringVar(&filterPorts, "port", "", "Filter by port(s): 3000, 3000-4000, :3000,:8000")
	rootCmd.PersistentFlags().StringVar(&filterFrameworks, "framework", "", "Filter by framework(s): nextjs,vite")
	rootCmd.PersistentFlags().StringVar(&filterProjects, "project", "", "Filter by project name/path: shop,~/projects/shop")
	rootCmd.PersistentFlags().StringVar(&filterUptime, "uptime", "", "Filter by uptime: >1h, <30m, 2h")
	rootCmd.Flags().BoolP("version", "v", false, "Print version")
	rootCmd.SetVersionTemplate("ports {{.Version}}\n")

	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	cfg := buildInspectConfig()
	services, err := inspect.Scan(cfg)
	if err != nil {
		return err
	}

	filter, err := inspect.ParseFilter(filterPorts, filterFrameworks, filterProjects, filterUptime)
	if err != nil {
		return err
	}
	services = inspect.ApplyFilter(services, filter)

	return writeReport(services)
}

func buildInspectConfig() *inspect.Config {
	var cfg inspect.Config

	// Custom frameworks
	if viper.IsSet("frameworks") {
		var custom []inspect.FrameworkRule
		if err := viper.UnmarshalKey("frameworks", &custom); err == nil {
			cfg.CustomFrameworks = custom
		}
	}

	// Custom project markers
	if viper.IsSet("projects.markers") {
		var markers []string
		if err := viper.UnmarshalKey("projects.markers", &markers); err == nil {
			cfg.CustomProjectMarkers = markers
		}
	}

	return &cfg
}
