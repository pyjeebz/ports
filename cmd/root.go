package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

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
	watch            bool
	watchInterval    time.Duration
	filterPorts      string
	filterFrameworks string
	filterProjects   string
	filterUptime     string
	sortBy           string
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

	// Load config file. A file that exists but doesn't parse is a bad
	// invocation — the user asked ports to run with input it can't read.
	if err := loadConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "ports: config:", err)
		os.Exit(2)
	}

	codifyExitCodes()

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

// Root exposes the built command tree. It exists for tools/genman, which
// renders the man pages at release time — same tree, same flags, same
// descriptions the binary ships with.
func Root() *cobra.Command { return rootCmd }

// codifyExitCodes makes every unusable invocation exit 2 — the grammar
// doctor already promised, uniform across commands. Cobra reports bad
// flags and bad argument counts as plain errors, which would exit 1, so
// wrap each command's validators and flag parsing in the code. What
// remains a plain error is a genuine runtime failure, and stays 1.
func codifyExitCodes() {
	toBadArgs := func(err error) error {
		return exitError{code: 2, msg: err.Error()}
	}
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return toBadArgs(err)
	})
	for _, c := range rootCmd.Commands() {
		sub := c
		sub.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
			return toBadArgs(err)
		})
		if orig := sub.Args; orig != nil {
			sub.Args = func(cmd *cobra.Command, args []string) error {
				if err := orig(cmd, args); err != nil {
					return toBadArgs(err)
				}
				return nil
			}
		}
	}

	// bare `ports` still lists; a stray positional is a command that
	// doesn't exist, and says so with exit 2 like every other bad arg
	rootCmd.Args = func(_ *cobra.Command, args []string) error {
		if len(args) > 0 {
			return exitError{code: 2, msg: fmt.Sprintf("unknown command %q — run `ports --help` for the list", args[0])}
		}
		return nil
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
	rootCmd.PersistentFlags().StringVar(&format, "format", "table", "Output format: table, json, snapshot, csv, tsv")
	rootCmd.PersistentFlags().BoolVar(&showTree, "tree", false, "Nest child processes under their parent (table format only)")
	rootCmd.PersistentFlags().StringVar(&filterPorts, "port", "", "Filter by port(s): 3000, 3000-4000, :3000,:8000")
	rootCmd.PersistentFlags().StringVar(&filterFrameworks, "framework", "", "Filter by framework(s): nextjs,vite")
	rootCmd.PersistentFlags().StringVar(&filterProjects, "project", "", "Filter by project name/path: shop,~/projects/shop")
	rootCmd.PersistentFlags().StringVar(&filterUptime, "uptime", "", "Filter by uptime: >1h, <30m, 2h")
	rootCmd.PersistentFlags().StringVar(&sortBy, "sort", "", "Sort order: port, uptime, process, project (default: grouped by project)")
	rootCmd.PersistentFlags().BoolVarP(&watch, "watch", "w", false, "Live refresh: rescan and redraw until interrupted")
	rootCmd.PersistentFlags().DurationVar(&watchInterval, "interval", time.Second, "Rescan interval for --watch (min 250ms)")
	rootCmd.Flags().BoolP("version", "v", false, "Print version")
	rootCmd.SetVersionTemplate("ports {{.Version}}\n")

	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	if watch {
		return runWatch()
	}
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
	if noMatch(services, filter) {
		fmt.Println(noMatchMsg)
		return nil
	}

	return writeReport(services, sort)
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
