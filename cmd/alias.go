package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"text/tabwriter"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/pyjeebz/ports/internal/router"
	"github.com/pyjeebz/ports/internal/routes"
	"github.com/spf13/cobra"
)

var aliasRemove bool

var aliasCmd = &cobra.Command{
	Use:   "alias [name] [port]",
	Short: "Give a fixed name to a port ports can't name on its own",
	Long: `ports alias names a port for good — for a server ports doesn't recognise
as a web dev server, or one that runs outside any project:

  $ ports alias stripe 4242
  ✓ stripe.localhost → :4242

An alias wins over a name ports found itself, and routes whatever
listens on its port. It's kept until removed — across restarts and
reboots — and when nothing holds the port, the name says so.

  ports alias                     list the aliases
  ports alias stripe 4242         name :4242 stripe
  ports alias stripe --remove     drop it

--json lists the aliases as [{"name", "port"}].`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		aliases, err := routes.LoadAliases()
		if err != nil {
			return err
		}
		switch {
		case aliasRemove:
			if len(args) != 1 {
				return exitError{code: 2, msg: "--remove takes the name alone: ports alias <name> --remove"}
			}
			name := nameArg(args[0])
			if _, ok := aliases[name]; !ok {
				return exitError{code: 1, msg: fmt.Sprintf("no alias is called %s", name)}
			}
			delete(aliases, name)
			if err := routes.SaveAliases(aliases); err != nil {
				return err
			}
			fmt.Println(inspect.Mark("✓ removed "+name+".localhost", false))
			return nil
		case len(args) == 0:
			return listAliases(aliases)
		case len(args) == 1:
			return exitError{code: 2, msg: fmt.Sprintf("give the port too: ports alias %s <port>", args[0])}
		}

		name := nameArg(args[0])
		if !routes.ValidName(name) {
			return exitError{code: 2, msg: fmt.Sprintf("%q can't be a name — use letters, digits and hyphens, with dots between parts (api.shop)", args[0])}
		}
		port, err := requirePort(args[1])
		if err != nil {
			return err
		}
		prev, had := aliases[name]
		aliases[name] = port
		if err := routes.SaveAliases(aliases); err != nil {
			return err
		}
		line := fmt.Sprintf("✓ %s → :%d", aliasURL(name), port)
		if had && prev != port {
			line += fmt.Sprintf(" (was :%d)", prev)
		}
		fmt.Println(inspect.Mark(line, false))
		return nil
	},
}

func init() {
	aliasCmd.Flags().BoolVar(&aliasRemove, "remove", false, "Drop the alias")
	aliasCmd.Flags().BoolVar(&jsonOut, "json", false, "JSON output")
	rootCmd.AddCommand(aliasCmd)
}

func aliasURL(name string) string {
	if info, ok := router.Running(); ok {
		return router.URL(name, info.Port)
	}
	return name + ".localhost"
}

func listAliases(aliases routes.Aliases) error {
	names := make([]string, 0, len(aliases))
	for n := range aliases {
		names = append(names, n)
	}
	slices.Sort(names)
	if jsonOut {
		type row struct {
			Name string `json:"name"`
			Port uint32 `json:"port"`
		}
		rows := []row{}
		for _, n := range names {
			rows = append(rows, row{n, aliases[n]})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	}
	if len(names) == 0 {
		fmt.Println("no aliases — `ports alias <name> <port>` adds one")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	for _, n := range names {
		_, _ = fmt.Fprintf(w, "%s\t:%d\n", aliasURL(n), aliases[n])
	}
	return w.Flush()
}
