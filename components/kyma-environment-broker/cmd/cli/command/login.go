package command

import (
	"errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/logger"
	"github.com/spf13/cobra"
)

// LoginCommand represents an execution of the kcp login command
type LoginCommand struct {
	log      logger.Logger
	username string
	password string
}

// NewLoginCmd constructs a new instance of LoginCommand and configures it in terms of a cobra.Command
func NewLoginCmd(log logger.Logger) *cobra.Command {
	cmd := LoginCommand{log: log}
	cobraCmd := &cobra.Command{
		Use:     "login",
		Aliases: []string{"l"},
		Short:   "Perform OIDC login required by all commands",
		Long: `Initiates OIDC login to obtain ID token, which is required by all CLI commands.
By default without any options, the OIDC authorization code flow is executed, which prompts the user to navigate to a local address in the browser and get redirected to the OIDC Authentication Server login page.
Service accounts can execute the resource owner credentials flow by specifying the --username and --password options.`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(cobraCmd *cobra.Command, _ []string) error { return cmd.Run(cobraCmd) },
	}
	cobraCmd.Flags().StringVarP(&cmd.username, "username", "u", "", "Username to use for resource owner credentials flow")
	cobraCmd.Flags().StringVarP(&cmd.password, "password", "p", "", "Password to use for resource owner credentials flow")

	return cobraCmd
}

// Run executes the login command
func (cmd *LoginCommand) Run(cobraCmd *cobra.Command) error {
	cred := CLICredentialManager(cmd.log)
	var err error
	if cmd.username == "" {
		_, err = cred.GetTokenByAuthCode(cobraCmd.Context())
	} else {
		_, err = cred.GetTokenByROPC(cobraCmd.Context(), cmd.username, cmd.password)
	}

	if err != nil {
		return err
	}
	return nil
}

// Validate checks the input parameters of the login command
func (cmd *LoginCommand) Validate() error {
	if cmd.username != "" && cmd.password == "" || cmd.username == "" && cmd.password != "" {
		return errors.New("both username and password must be specified for resource owner credentials login")
	}
	return nil
}
