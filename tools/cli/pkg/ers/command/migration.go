package command

import "github.com/spf13/cobra"

func NewMigrationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "migrate",
		Short:   "Triggers SC migration.",
		Long: `Triggers SC migration.`,
	}

	return cmd
}
