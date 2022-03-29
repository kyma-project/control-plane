package command

import (
	"errors"
	"fmt"

	"github.com/kyma-project/control-plane/tools/cli/pkg/ers/client"
	"github.com/spf13/cobra"
)

type SwitchCommand struct {
	cobraCmd *cobra.Command
	brokerId string
}

func NewSwitchBrokerCommand() *cobra.Command {
	cmd := &SwitchCommand{}
	cobraCmd := &cobra.Command{
		Use:   "switch [id]",
		Short: "Switching a broker to SM operator credentials.",
		Long: `The command use "/provisioning/v1/kyma/brokers/{brokerId}" endpoint to switch to SM operator credentials.
The broker is specified by an ID`,
		Example: `ers switch abcd-1234		switches broker abcd-1234 to use SM operator credentials.`,
		PreRunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return errors.New("Missing required param `id`")
			}

			cmd.brokerId = args[0]
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run()
		},
	}

	cmd.cobraCmd = cobraCmd

	return cobraCmd
}

func (c *SwitchCommand) Run() error {
	ers, err := client.NewErsClient()
	if err != nil {
		return fmt.Errorf("while initializing ers client: %w", err)
	}
	defer ers.Close()
	// TODO: Correcte responses

	return ers.Migrate(c.brokerId)
}
