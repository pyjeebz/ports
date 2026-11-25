package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"

	"golang.org/x/term"

	"github.com/pyjeebz/ports/internal/hosts"
	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/pyjeebz/ports/internal/routes"
	"github.com/pyjeebz/ports/internal/service"
	"github.com/spf13/cobra"
)

var (
	hostsPrune bool
	hostsFile  string
)

var hostsCmd = &cobra.Command{
	Use:   "hosts",
	Short: "Write dev server names to the hosts file, for clients that need it",
	Long: `Browsers and curl send every *.localhost name to this machine on their
own. Node's fetch, Python and Go ask the system resolver instead, which
on Windows — and on a Linux whose resolv.conf skips systemd-resolved's
stub — doesn't know shop.localhost. ports hosts sync writes the names
into /etc/hosts so those clients find them too.

  ports hosts              what the hosts file holds, and what it lacks
  ports hosts sync         write the running names and aliases there
  ports hosts clean        take them all out

The names live in one marked block; nothing else in the file is
touched. A sync keeps the names already there — a stopped server's name
still resolves for when it's back — unless --prune. Changing the hosts
file needs root: sync and clean ask sudo, once each time, and run
nothing else as root. On Windows, run them from an elevated terminal.
On WSL, the distro rewrites /etc/hosts at boot unless generateHosts is
off in /etc/wsl.conf.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error { return hostsShow() },
}

var hostsSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Write the running names and aliases into the hosts file",
	Args:  cobra.NoArgs,
	RunE:  func(cmd *cobra.Command, args []string) error { return hostsSync() },
}

var hostsCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Take every name ports wrote out of the hosts file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return hostsSet(nil, "✓ the hosts file holds no names from ports")
	},
}

// hostsApplyCmd is the one step that runs as root: it reads names from
// stdin, refuses any that isn't a name, and writes the block. sync and
// clean run it under sudo, so root never scans, names or reads state.
var hostsApplyCmd = &cobra.Command{
	Use:    "apply",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var names []string
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			name := strings.TrimSpace(sc.Text())
			if name == "" {
				continue
			}
			if !routes.ValidName(name) {
				return exitError{code: 2, msg: fmt.Sprintf("%q is not a name", name)}
			}
			names = append(names, name)
		}
		if err := sc.Err(); err != nil {
			return err
		}
		return hosts.Write(hostsFile, names)
	},
}

func init() {
	hostsCmd.Flags().BoolVar(&jsonOut, "json", false, "JSON output")
	hostsSyncCmd.Flags().BoolVar(&hostsPrune, "prune", false, "Drop names that no server or alias holds now")
	hostsApplyCmd.Flags().StringVar(&hostsFile, "file", hosts.Path(), "The hosts file")
	hostsCmd.AddCommand(hostsSyncCmd, hostsCleanCmd, hostsApplyCmd)
	rootCmd.AddCommand(hostsCmd)
}

// currentNames is every name that should resolve now: the routable ones
// running, and the aliases.
func currentNames() ([]string, error) {
	services, err := scanNamed(buildInspectConfig())
	if err != nil {
		return nil, err
	}
	var names []string
	for _, s := range services {
		if routes.Routable(s) && !slices.Contains(names, s.Name) {
			names = append(names, s.Name)
		}
	}
	if aliases, err := routes.LoadAliases(); err == nil {
		for n := range aliases {
			if !slices.Contains(names, n) {
				names = append(names, n)
			}
		}
	}
	slices.Sort(names)
	return names, nil
}

func hostsShow() error {
	path := hosts.Path()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	held := hosts.Names(data)
	current, err := currentNames()
	if err != nil {
		return err
	}
	var missing []string
	for _, n := range current {
		if !slices.Contains(held, n) {
			missing = append(missing, n)
		}
	}
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			File    string   `json:"file"`
			Names   []string `json:"names"`
			Missing []string `json:"missing"`
		}{path, nonNil(held), nonNil(missing)})
	}
	if len(held) == 0 {
		fmt.Printf("%s holds no names from ports\n", path)
	} else {
		fmt.Printf("%s holds %d name(s): %s\n", path, len(held), strings.Join(held, ", "))
	}
	if len(missing) > 0 {
		fmt.Printf("not there yet: %s — `ports hosts sync` adds them\n", strings.Join(missing, ", "))
	}
	return nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func hostsSync() error {
	names, err := currentNames()
	if err != nil {
		return err
	}
	if !hostsPrune {
		if data, err := os.ReadFile(hosts.Path()); err == nil {
			for _, n := range hosts.Names(data) {
				if !slices.Contains(names, n) {
					names = append(names, n)
				}
			}
			slices.Sort(names)
		}
	}
	if len(names) == 0 {
		fmt.Println("no names to write — nothing has one yet")
		return nil
	}
	return hostsSet(names, fmt.Sprintf("✓ %s resolves %d name(s) for every client: %s", hosts.Path(), len(names), strings.Join(names, ", ")))
}

// hostsSet makes the block hold exactly names: directly when this process
// may write the file, else through sudo running ports hosts apply.
func hostsSet(names []string, done string) error {
	path := hosts.Path()
	if data, err := os.ReadFile(path); err == nil && slices.Equal(hosts.Names(data), names) {
		fmt.Println(inspect.Mark(done, false))
		return nil
	}
	if hosts.Writable(path) {
		if err := hosts.Write(path, names); err != nil {
			return err
		}
		fmt.Println(inspect.Mark(done, false))
		return nil
	}
	if runtime.GOOS == "windows" {
		return exitError{code: 1, msg: fmt.Sprintf("%s needs an administrator — run this from an elevated terminal (Run as administrator)", path)}
	}
	exe, err := service.Executable()
	if err != nil {
		return err
	}
	args := []string{exe, "hosts", "apply", "--file", path}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// no one to type a password: fail at once rather than hang
		args = append([]string{"-n"}, args...)
	} else {
		fmt.Fprintf(os.Stderr, "%s belongs to root — sudo asks to write the ports block, and nothing else:\n", path)
	}
	sudo := exec.Command("sudo", args...)
	sudo.Stdin = strings.NewReader(strings.Join(names, "\n") + "\n")
	sudo.Stdout, sudo.Stderr = os.Stdout, os.Stderr
	if err := sudo.Run(); err != nil {
		return exitError{code: 1, msg: fmt.Sprintf("the hosts file wasn't changed (%v) — run `ports hosts sync` in a terminal, where sudo can ask", err)}
	}
	fmt.Println(inspect.Mark(done, false))
	return nil
}
