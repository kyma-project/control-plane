package command

import "github.com/spf13/cobra"

func NewSwitchBrokerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch-broker",
		Short: "Switching a broker to SM operator credentials.",
		Long: `The command use "/provisioning/v1/kyma/brokers/{brokerId}" endpoint to switch to SM operator credentials.
The broker is specified by an ID`,
		Example: `ers switch-broker abcd-1234		switches broker abcd-1234 to use SM operator credentials.`,
	}

	return cmd
}
