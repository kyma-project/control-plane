package command

import (
	"fmt"

	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers/client"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers/fetcher"
	"github.com/kyma-project/control-plane/tools/cli/pkg/printer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

type Printer interface {
	PrintObj(o interface{}) error
}

var tableColumns = []printer.Column{
	{
		Header:    "INSTANCE ID",
		FieldSpec: "{.Id}",
	},
	{
		Header:    "GLOBALACCOUNT ID",
		FieldSpec: "{.GlobalAccountID}",
	},
	{
		Header:    "BROKER ID",
		FieldSpec: "{.BrokerId}",
	},
	{
		Header:    "MIGRATED",
		FieldSpec: "{.Migrated}",
	},
}

type Filters struct {
	GlobalAccountID string
	Migrated        bool
	NotMigrated     bool
	InstanceID      string

	// TODO: define more filters
}

type InstancesCommand struct {
	cobraCmd        *cobra.Command
	instanceFetcher fetcher.InstanceFetcher
	filters         Filters
	source          string
	pageStart       int
	pageSize        int
	pageLimit       int
	output          string
}

func (c *InstancesCommand) Run() error {

	if c.source != "" {
		c.instanceFetcher = fetcher.NewFileClient(c.source)
	} else {

		ers, err := client.NewErsClient(ers.GlobalOpts.ErsUrl())
		if err != nil {
			return errors.Wrap(err, "while initializing ers client")
		}
		defer ers.Close()

		// todo: use real client to ers
		c.instanceFetcher = fetcher.NewInitialFetcher(ers, c.pageStart, c.pageSize, c.pageLimit)
	}

	var pr Printer
	switch c.output {
	case jsonOutput:
		pr = printer.NewJSONPrinter("  ")
	case tableOutput:
		pr, _ = printer.NewTablePrinter(tableColumns, false)
	}

	if c.filters.InstanceID != "" {
		instance, err := c.instanceFetcher.GetInstanceById(c.filters.InstanceID)
		pr.PrintObj(instance)
		return err
	}
	var result []ers.Instance
	instances, err := c.instanceFetcher.GetAllInstances()
	for _, item := range instances {
		if c.filters.Migrated && !item.Migrated {
			continue
		}
		if c.filters.NotMigrated && item.Migrated {
			continue
		}
		if c.filters.GlobalAccountID != "" && item.GlobalAccountID != c.filters.GlobalAccountID {
			continue
		}
		result = append(result, item)
	}

	pr.PrintObj(result)
	return err
}

func (c *InstancesCommand) Validate() error {
	if c.output != tableOutput && c.output != jsonOutput {
		return fmt.Errorf("invalid value for output: %s", c.output)
	}
	return nil
}

func NewInstancesCommand(log *logrus.Logger) *cobra.Command {
	cmd := &InstancesCommand{}
	corbaCmd := &cobra.Command{
		Use:   "instances",
		Short: "Displays ERS instances.",
		Long:  `Displays information about ERS instances.`,
		Example: `  ers instances -i fff090a1-b46f-4f14-a79e-681b00227921		Display details about the instance fff090a1-b46f-4f14-a79e-681b00227921.
  ers instances -g 0f9a6a13-796b-4b6e-ac22-0d1512261a83		Display details about all instances of a given global account
  ers instances -s 0f9a6a13-796b-4b6e-ac22-0d1512261a83		Display details about all instances of a given subaccount
  ers instances -s 44534cae-796b-c46a-ef52-aa15125a023b		Display details about all instances created by given broker`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run()
		},
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
	}

	cmd.cobraCmd = corbaCmd

	corbaCmd.Flags().BoolVar(&cmd.filters.Migrated, "migrated", false, "Get migrated instances")
	corbaCmd.Flags().BoolVar(&cmd.filters.NotMigrated, "not-migrated", false, "Get not migrated instances")
	corbaCmd.Flags().StringVarP(&cmd.filters.InstanceID, "instance-id", "i", "", "Get not migrated instances")
	corbaCmd.Flags().StringVarP(&cmd.filters.GlobalAccountID, "global-account-id", "g", "", "Filter by global account ID.")
	corbaCmd.Flags().StringVar(&cmd.source, "source", "", "File containing instances data")
	corbaCmd.Flags().IntVar(&cmd.pageStart, "pageNo", 0, "Specify which page to load")
	corbaCmd.Flags().IntVar(&cmd.pageSize, "pageSize", 0, "Specify how many elements per page to load")
	corbaCmd.Flags().IntVar(&cmd.pageLimit, "pageLimit", 0, "Specify how many pages to load, by default loads all")
	corbaCmd.Flags().StringVarP(&cmd.output, "output", "o", tableOutput, fmt.Sprintf("Output type of displayed Instances. The possible values are: %s, %s.", tableOutput, jsonOutput))

	return corbaCmd
}
