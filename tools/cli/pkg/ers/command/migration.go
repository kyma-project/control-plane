package command

import (
	"errors"
	"fmt"

	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers/client"
	"github.com/spf13/cobra"
)

func NewMigrationCommand() *cobra.Command {
	cmd := &MigrationCommand{}

	cobraCmd := &cobra.Command{
		Use:   "migrate [id]",
		Short: "Triggers SC migration.",
		Long:  `Migrates a single environment instance from the sm credentials to the operator-based credentials. The migration is run for not migrated instances.`,
		Example: `  ers migrate									Triggers a migration for all not migrated instances
ers migrate -g 0f9a6a13-796b-4b6e-ac22-0d1512261a83		Triggers a migration for all instances of a given global account
ers migrate --source input.json		Triggers a migration for all instances using instance date stored in a file.`,
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return errors.New("Missing required param `id`")
			}

			cmd.instanceID = args[0]
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run()
		},
	}

	cobraCmd.Flags().StringVarP(&cmd.instanceID, "instance-id", "i", "", "Get not migrated instances")

	cmd.corbaCmd = cobraCmd

	// TODO: we should not run migration for failed instances (to be clarified)

	return cobraCmd
}

type MigrationCommand struct {
	corbaCmd   *cobra.Command
	instanceID string
}

func (c *MigrationCommand) Run() error {
	ers, err := client.NewErsClient(ers.GlobalOpts.ErsUrl())
	if err != nil {
		return fmt.Errorf("while initializing ers client: %w", err)
	}
	defer ers.Close()

	return ers.Migrate(c.instanceID)
}
