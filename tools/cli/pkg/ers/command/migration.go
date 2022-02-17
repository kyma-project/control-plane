package command

import "github.com/spf13/cobra"

func NewMigrationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Triggers SC migration.",
		Long:  `Migrates a single environment instance from the sm credentials to the operator-based credentials. The migration is run for not migrated instances.`,
		Example: `  ers migrate									Triggers a migration for all not migrated instances
  ers migrate -g 0f9a6a13-796b-4b6e-ac22-0d1512261a83		Triggers a migration for all instances of a given global account
  ers migrate --source input.json		Triggers a migration for all instances using instance date stored in a file.`,
	}

	// TODO: we should not run migration for failed instances (to be clarified)

	return cmd
}
