package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

var hereCmd = &cobra.Command{
	Use:   "here [dir]",
	Short: "What's running for this project — the one you're in, or the one named",
	Long: `ports here answers "is this project already running?" — the question to
ask before starting a second dev server on the next free port.

  $ cd ~/projects/shop && ports here
  PORT   PROCESS      PROJECT              UPTIME
  3000   Next.js      ~/projects/shop      14m
  8000   FastAPI      ~/projects/shop      14m
  5432   PostgreSQL   docker: shop-db      2d

The project is found the way the PROJECT column finds it — the outermost
.git, then workspace markers, then the nearest package.json, go.mod, and
friends — so any subdirectory works. Compose containers belong when they
were started from inside the project.

Exit codes: 0 when something is running, 1 when nothing is (grep-style,
so ` + "`ports here --port 3000 || npm run dev`" + ` starts a server only when
none is up), 2 for a bad invocation. --format json prints [] when nothing
is running, never empty output.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// validate the arguments before paying for a scan
		sort, err := resolveSort()
		if err != nil {
			return err
		}
		filter, err := inspect.ParseFilter(filterPorts, filterFrameworks, filterProjects, filterUptime)
		if err != nil {
			return exitError{code: 2, msg: err.Error()}
		}
		dir := "."
		if len(args) == 1 {
			dir = args[0]
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			return exitError{code: 2, msg: err.Error()}
		}
		if st, err := os.Stat(abs); err != nil || !st.IsDir() {
			return exitError{code: 2, msg: fmt.Sprintf("%s is not a directory", dir)}
		}

		cfg := buildInspectConfig()
		// resolved first, so the root is found from the same path the
		// PROJECT column walks up from — the kernel's, links resolved
		root := inspect.ProjectRoot(cfg, inspect.RealPath(abs))
		services, err := inspect.Scan(cfg)
		if err != nil {
			return err
		}
		matched := inspect.ApplyFilter(inspect.Here(services, root), filter)
		if len(matched) == 0 {
			return noneFound(fmt.Sprintf("nothing running for %s — `ports` shows the whole machine", inspect.ShortPath(root)))
		}
		return writeReport(matched, sort)
	},
}

func init() {
	formatFlags(hereCmd, allFormats)
	tableFlags(hereCmd)
	filterFlags(hereCmd)
	rootCmd.AddCommand(hereCmd)
}
