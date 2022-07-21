package command

import (
	"errors"

	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// LoginCommand represents an execution of the kcp login command
type LoginCommand struct {
	cobraCmd *cobra.Command
	log      logger.Logger
	username string
	password string
}

// NewLoginCmd constructs a new instance of LoginCommand and configures it in terms of a cobra.Command
func NewLoginCmd() *cobra.Command {
	cmd := LoginCommand{}
	cobraCmd := &cobra.Command{
		Use:     "loginn",
		Aliases: []string{"l"},
		Short:   "Performs OIDC login required by all commands.",
		Long: `Initiates OIDC login to obtain the ID token which is required by all CLI commands.
By default, without any options, the OIDC authorization code flow is executed. It prompts the user to navigate to a local address in the browser and get redirected to the OIDC Authentication Server login page.
Service accounts can execute the resource owner credentials flow by specifying the --username and --password options.`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd
	cobraCmd.Flags().StringVarP(&cmd.username, "username", "u", "", "Username to use for the resource owner credentials flow.")
	cobraCmd.Flags().StringVarP(&cmd.password, "password", "p", "", "Password to use for the resource owner credentials flow.")

	return cobraCmd
}

// Run executes the login command
func (cmd *LoginCommand) Run() error {
	cmd.log = logger.New()
	cred := CLICredentialManager(cmd.log)
	var err error
	if cmd.username == "" {
		_, err = cred.GetTokenByAuthCode(cmd.cobraCmd.Context())
	} else {
		_, err = cred.GetTokenByROPC(cmd.cobraCmd.Context(), cmd.username, cmd.password)
	}
	if err != nil {
		return err
	}

	viper.Set(GlobalOpts.username, cmd.username)

	return err
}

// Validate checks the input parameters of the login command
func (cmd *LoginCommand) Validate() error {
	if cmd.username != "" && cmd.password == "" || cmd.username == "" && cmd.password != "" {
		return errors.New("both username and password must be specified for resource owner credentials login")
	}
	return nil
}
