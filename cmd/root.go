package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	jsonOut          bool
	force            bool
	dryRun           bool
	expect           string
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
	flagErr := func(c *cobra.Command, err error) error {
		return toBadArgs(misplacedFlag(c, err))
	}
	rootCmd.SetFlagErrorFunc(flagErr)
	for _, c := range rootCmd.Commands() {
		sub := c
		sub.SetFlagErrorFunc(flagErr)
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

// misplacedFlag turns "unknown flag" into where the flag does apply,
// when another command owns it: `ports kill --watch` learns --watch is
// a listing flag, not that it doesn't exist.
func misplacedFlag(c *cobra.Command, err error) error {
	msg := err.Error()
	var lookup func(*cobra.Command) *pflag.Flag
	var shown string
	if name, ok := strings.CutPrefix(msg, "unknown flag: --"); ok {
		lookup = func(o *cobra.Command) *pflag.Flag { return o.Flags().Lookup(name) }
		shown = "--" + name
	} else if rest, ok := strings.CutPrefix(msg, "unknown shorthand flag: '"); ok && rest != "" {
		short := rest[:1]
		lookup = func(o *cobra.Command) *pflag.Flag { return o.Flags().ShorthandLookup(short) }
		shown = "-" + short
	} else {
		return err
	}
	var owners []string
	for _, o := range append([]*cobra.Command{rootCmd}, rootCmd.Commands()...) {
		if o != c && lookup(o) != nil {
			owners = append(owners, "`"+o.CommandPath()+"`")
		}
	}
	if len(owners) == 0 {
		return err
	}
	return fmt.Errorf("%s doesn't apply to `%s` — it's a flag of %s", shown, c.CommandPath(), strings.Join(owners, ", "))
}

// scanConfig is the optional ports-config.yaml, read once at startup.
var scanConfig inspect.Config

// loadConfig reads ports-config.yaml from the first place it exists. No
// file is the normal case; a file that exists but doesn't parse — or
// whose values don't fit — fails the run rather than be half-applied.
func loadConfig() error {
	v := viper.New()
	v.SetConfigName("ports-config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.config/ports")
	v.AddConfigPath("/etc/ports")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil
		}
		// three places are searched — say which file is broken
		if where := v.ConfigFileUsed(); where != "" {
			return fmt.Errorf("%s: %w", where, err)
		}
		return err
	}
	where := v.ConfigFileUsed()
	if err := v.UnmarshalKey("frameworks", &scanConfig.CustomFrameworks); err != nil {
		return fmt.Errorf("%s: frameworks: %w", where, err)
	}
	if err := v.UnmarshalKey("projects.markers", &scanConfig.CustomProjectMarkers); err != nil {
		return fmt.Errorf("%s: projects.markers: %w", where, err)
	}
	if err := scanConfig.Validate(); err != nil {
		return fmt.Errorf("%s: %w", where, err)
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
	listingFlags(rootCmd)
	rootCmd.Flags().BoolP("version", "v", false, "Print version")
	rootCmd.SetVersionTemplate("ports {{.Version}}\n")

	listingFlags(listCmd)
	rootCmd.AddCommand(listCmd)
}

// Each command carries the flags it honors and no others: a flag a
// command would ignore is an unknown flag there — exit 2, naming the
// commands it belongs to — never a silent no-op.

// allFormats is every render the listing commands can draw.
const allFormats = "table, json, snapshot, csv, tsv"

func formatFlags(c *cobra.Command, formats string) {
	c.Flags().StringVar(&format, "format", "table", "Output format: "+formats)
	c.Flags().BoolVar(&jsonOut, "json", false, "JSON output, same as --format json")
}

func filterFlags(c *cobra.Command) {
	c.Flags().StringVar(&filterPorts, "port", "", "Filter by port(s): 3000, 3000-4000, :3000,:8000")
	c.Flags().StringVar(&filterFrameworks, "framework", "", "Filter by framework(s): nextjs,vite")
	c.Flags().StringVar(&filterProjects, "project", "", "Filter by project name/path: shop,~/projects/shop")
	c.Flags().StringVar(&filterUptime, "uptime", "", "Filter by uptime: >1h, <30m, 2h")
}

func tableFlags(c *cobra.Command) {
	c.Flags().StringVar(&sortBy, "sort", "", "Sort order: port, uptime, process, project (default: grouped by project)")
	c.Flags().BoolVar(&showTree, "tree", false, "Nest child processes under their parent (table format only)")
}

func listingFlags(c *cobra.Command) {
	formatFlags(c, allFormats)
	tableFlags(c)
	filterFlags(c)
	c.Flags().BoolVarP(&watch, "watch", "w", false, "Live refresh: rescan and redraw until interrupted")
	c.Flags().DurationVar(&watchInterval, "interval", time.Second, "Rescan interval for --watch (min 250ms)")
}

func stopFlags(c *cobra.Command) {
	formatFlags(c, "table, json")
	c.Flags().BoolVar(&force, "force", false, "SIGKILL immediately instead of SIGTERM first")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be stopped without stopping it")
	c.Flags().StringVar(&expect, "expect", "", "Stop only if everything matched is this project, framework, container, or process — else stop nothing")
}

// stopOptions gathers the stop flags for stopAll.
func stopOptions() stopOpts {
	return stopOpts{force: force, dryRun: dryRun, expect: expect}
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

// buildInspectConfig hands a scan its own copy of the loaded config.
func buildInspectConfig() *inspect.Config {
	cfg := scanConfig
	return &cfg
}
