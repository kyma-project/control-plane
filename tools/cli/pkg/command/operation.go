package command

import (
	"context"

	"github.com/spf13/cobra"
)

type operationCmd struct {
	ctx context.Context
}

func NewOperationCmd() *cobra.Command {
	cmd := operationCmd{}
	cobraCmd := &cobra.Command{
		Use:   "operation",
		Short: "Manage operations",
	}

	cobraCmd.AddCommand(
		NewOperationStopCmd(),
	)

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
	}

	cmd.ctx = context.Background()
	return cobraCmd
}
