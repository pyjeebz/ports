package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `Generate shell completion scripts for ports.

To load completions:

Bash:
  $ source <(ports completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ ports completion bash > /etc/bash_completion.d/ports
  # macOS:
  $ ports completion bash > /usr/local/etc/bash_completion.d/ports

Zsh:
  $ source <(ports completion zsh)

  # To load completions for each session, execute once:
  $ ports completion zsh > "${fpath[1]}/_ports"

Fish:
  $ ports completion fish | source

  # To load completions for each session, execute once:
  $ ports completion fish > ~/.config/fish/completions/ports.fish

PowerShell:
  $ ports completion powershell | Out-String | Invoke-Expression

  # To load completions for each session, add to your profile:
  $ ports completion powershell >> $PROFILE.CurrentUserAllHosts
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletion(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
