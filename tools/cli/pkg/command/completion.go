package command

import (
	"os"

	"github.com/spf13/cobra"
)

// CompletionCommand represents an execution of the kcp completion command
type CompletionCommand struct {
	cobraCmd *cobra.Command
	output   string
}

var validArgements = []string{"bash", "zsh", "fish", "powershell"}

// NewCompletionCmd constructs a new instance of CompletionCommand and configures it in terms of a cobra.Command
func NewCompletionCommand() *cobra.Command {
	cmd := CompletionCommand{}
	cobraCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `To load completions:

Bash:

$ source <(kcp completion bash)

# To load completions for each session, execute once:
Linux:
  $ kcp completion bash > /etc/bash_completion.d/kcp
MacOS:
  $ kcp completion bash > /usr/local/etc/bash_completion.d/kcp

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ kcp completion zsh > "${fpath[1]}/_kcp"

# You will need to start a new shell for this setup to take effect.

Fish:

$ kcp completion fish | source

# To load completions for each session, execute once:
$ kcp completion fish > ~/.config/fish/completions/kcp.fish

Powershell:

PS> kcp completion powershell | Out-String | Invoke-Expression

# To load completions for every new session, run:
PS> kcp completion powershell > kcp.ps1
# and source this file from your powershell profile.
`,
		Example:               `kcp completion bash                            Display completions in bash.`,
		DisableFlagsInUseLine: true,
		ValidArgs:             validArgements,
		Args:                  cobra.ExactValidArgs(1),
		PreRunE:               func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:                  func(comd *cobra.Command, args []string) error { return cmd.Run(comd, args) },
	}
	SetOutputOpt(cobraCmd, &cmd.output)

	return cobraCmd
}

// Run executes the completion command
func (cmd *CompletionCommand) Run(comd *cobra.Command, args []string) error {
	//var comd *cobra.Command
	switch args[0] {
	case "bash":
		comd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		comd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		comd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		comd.Root().GenPowerShellCompletion(os.Stdout)
	}
	return nil
}

// Validate checks the input parameters of the runtimes command
func (cmd *CompletionCommand) Validate() error {
	err := ValidateOutputOpt(cmd.output)
	if err != nil {
		return err
	}
	return nil
}
