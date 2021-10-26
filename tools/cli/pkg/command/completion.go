package command

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// CompletionCommand represents an execution of the kcp completion command
type CompletionCommand struct {
	completeTarget string
}

var validArguements = []string{"bash", "zsh", "fish", "powershell"}

const errorMsg = "accepts 1 arg(s), received 0 \n"
const suggestionMsg = "Please use `kcp completion [bash|zsh|fish|powershell]`"
const savedInMsg = "Saved in"
const savedFileName = "kcp_completion"

// NewCompletionCmd constructs a new instance of CompletionCommand and configures it in terms of a cobra.Command
func NewCompletionCommand() *cobra.Command {
	cmd := CompletionCommand{}
	cobraCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generates completion script",
		Long: `Generates completion script for Bash, Zsh, Fish, or Powershell.
### Bash

To load completions for Bash, run:
` + "`$ source <(kcp completion bash)`" + `

To load completions for each session, execute once:

- Linux:
` + "`$ kcp completion bash > /etc/bash_completion.d/kcp`" + `

- MacOS:
` + "`$ kcp completion bash > /usr/local/etc/bash_completion.d/kcp`" + `

### Zsh

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:
` + "`$ echo \"autoload -U compinit; compinit\" >> ~/.zshrc`" + `

To load completions for each session, execute once:
` + "`$ kcp completion zsh > \"${fpath[1]}/_kcp\"`" + `

You will need to start a new shell for this setup to take effect.

### Fish

To load completions for Fish, run:
` + "`$ kcp completion fish | source`" + `

To load completions for each session, execute once:
` + "`$ kcp completion fish > ~/.config/fish/completions/kcp.fish`" + `

### Powershell

` + "`PS> kcp completion powershell | Out-String | Invoke-Expression`" + `

To load completions for every new session, run:
` + "`PS> kcp completion powershell > kcp.ps1`" + `

Source this file from your Powershell profile.
`,
		Example:               `kcp completion bash                            Display completions in bash.`,
		DisableFlagsInUseLine: true,
		ValidArgs:             validArguements,
		PreRunE:               func(_ *cobra.Command, args []string) error { return cmd.Validate(args) },
		RunE:                  func(comd *cobra.Command, args []string) error { return cmd.Run(comd, args) },
	}

	defaultOutputFile := setDefaultFile()
	cobraCmd.PersistentFlags().StringVarP(&cmd.completeTarget, "o", "", defaultOutputFile, "autocompletion file")

	return cobraCmd
}

// Run executes the completion command
func (cmd *CompletionCommand) Run(comd *cobra.Command, args []string) error {
	switch args[0] {
	case "bash":
		comd.Root().GenBashCompletionFile(cmd.completeTarget)
		fmt.Println(savedInMsg, cmd.completeTarget)
	case "zsh":
		comd.Root().GenZshCompletionFile(cmd.completeTarget)
		fmt.Println(savedInMsg, cmd.completeTarget)
	case "fish":
		comd.Root().GenFishCompletionFile(cmd.completeTarget, true)
		fmt.Println(savedInMsg, cmd.completeTarget)
	case "powershell":
		comd.Root().GenPowerShellCompletionFile(cmd.completeTarget)
		fmt.Println(savedInMsg, cmd.completeTarget)
	default:
		return errors.New(suggestionMsg)
	}
	return nil
}

// Validate checks the input parameters of the completion command
func (cmd *CompletionCommand) Validate(args []string) error {
	if len(args) == 0 {
		return errors.New(errorMsg + suggestionMsg)
	}
	return nil
}

// Set defaultFile for saving the completion files.
func setDefaultFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	return home + "/" + savedFileName
}
