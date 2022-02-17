package command

import (
	"encoding/json"
	"fmt"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers/client"
	"github.com/kyma-project/control-plane/tools/cli/pkg/printer"
	"github.com/spf13/cobra"
)

var tableColumns = []printer.Column{
	{
		Header:    "INSTANCE ID",
		FieldSpec: "{.Id}",
	},
	{
		Header:    "GLOBALACCOUNT ID",
		FieldSpec: "{.GlobalAccountId}",
	},
	{
		Header:    "MIGRATED",
		FieldSpec: "{.Migrated}",
	},
}

type InstanceFetcher interface {
	GetAllInstances() ([]ers.Instance, error)
	GetInstanceById(id string) (ers.Instance, error)
}

type Filters struct {
	GlobalAccountID string
	Migrated        bool
	NotMigrated     bool
	InstanceID      string

	// TODO: define more filters
}

type InstancesCommand struct {
	corbaCmd        *cobra.Command
	instanceFetcher InstanceFetcher
	filters         Filters
	source          string
}

func (c *InstancesCommand) Run() error {
	// TODO: this is a dummy implementation
	if c.source != "" {
		c.instanceFetcher = client.NewFileClient(c.source)
	} else {
		// todo: use real client to ers
		c.instanceFetcher = &emptyFetcher{}
	}

	if c.filters.InstanceID != "" {
		instance, _ := c.instanceFetcher.GetInstanceById(c.filters.InstanceID)
		b, _ := json.Marshal(instance)
		fmt.Println(string(b))
		return nil
	}
	var result []ers.Instance
	instances, _ := c.instanceFetcher.GetAllInstances()
	for _, item := range instances {
		if c.filters.Migrated && !item.Migrated {
			continue
		}
		if c.filters.NotMigrated && item.Migrated {
			continue
		}
		if c.filters.GlobalAccountID != "" && item.GlobalAccountId != c.filters.GlobalAccountID {
			continue
		}
		result = append(result, item)
	}

	//b, _ := json.Marshal(instances)

	tp, _ := printer.NewTablePrinter(tableColumns, false)
	tp.PrintObj(result)

	return nil
}

func NewInstancesCommand() *cobra.Command {
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
		//RunE: func(cmd *cobra.Command, args []string) error {
		//		// TODO: this is an example of a call to ERS
		//
		//		url := GlobalOpts.ErsUrl() + "provisioning/v1/kyma/environments?page=0&size=60"
		//
		//		resp, err := ErsHttpClient.Get(url)
		//		if err != nil {
		//			fmt.Println("Request error: ", err)
		//			return err
		//		}
		//		defer func() {
		//			resp.Body.Close()
		//		}()
		//
		//		d, err := ioutil.ReadAll(resp.Body)
		//		if err != nil {
		//			fmt.Println("Read error:", err)
		//			return err
		//		}
		//		fmt.Println(string(d))
		//		return nil
		//	},
	}

	cmd.corbaCmd = corbaCmd

	corbaCmd.Flags().BoolVar(&cmd.filters.Migrated, "migrated", false, "Get migrated instances")
	corbaCmd.Flags().BoolVar(&cmd.filters.NotMigrated, "not-migrated", false, "Get not migrated instances")
	corbaCmd.Flags().StringVarP(&cmd.filters.InstanceID, "instance-id", "i", "", "Get not migrated instances")
	corbaCmd.Flags().StringVar(&cmd.source, "source", "", "File containing instances data")

	return corbaCmd
}

type emptyFetcher struct {
}

func (e emptyFetcher) GetAllInstances() ([]ers.Instance, error) {
	panic("implement me")
}

func (e emptyFetcher) GetInstanceById(id string) (ers.Instance, error) {
	panic("implement me")
}
