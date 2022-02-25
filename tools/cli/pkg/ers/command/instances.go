package command

import (
	"context"
	"errors"

	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers/client"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers/fetcher"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/kyma-project/control-plane/tools/cli/pkg/printer"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/clientcredentials"

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
	corbaCmd        *cobra.Command
	instanceFetcher fetcher.InstanceFetcher
	filters         Filters
	source          string
	pageStart       int
	pageSize        int
	pageLimit       int
}

func (c *InstancesCommand) Run() error {
	if GlobalOpts.ClientID() == "" ||
		GlobalOpts.ClientSecret() == "" ||
		GlobalOpts.OauthUrl() == "" {
		return errors.New("no auth data provided")
	}

	config := clientcredentials.Config{
		ClientID:     GlobalOpts.ClientID(),
		ClientSecret: GlobalOpts.ClientSecret(),
		TokenURL:     GlobalOpts.OauthUrl(),
	}
	configClient := config.Client(context.Background())

	if c.source != "" {
		c.instanceFetcher = fetcher.NewFileClient(c.source)
	} else {
		// create a shared ERS HTTP client which does the oauth flow
		httpClient := client.NewHTTPClient(logger.New(), configClient)
		ers := client.NewErsClient(GlobalOpts.ErsUrl(), httpClient)

		// todo: use real client to ers
		c.instanceFetcher = fetcher.NewInitialFetcher(ers, c.pageStart, c.pageSize, c.pageLimit)
	}

	tp, _ := printer.NewTablePrinter(tableColumns, false)

	if c.filters.InstanceID != "" {
		instance, err := c.instanceFetcher.GetInstanceById(c.filters.InstanceID)
		tp.PrintObj(instance)
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

	configClient.CloseIdleConnections()

	tp.PrintObj(result)
	return err
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
	}

	cmd.corbaCmd = corbaCmd

	corbaCmd.Flags().BoolVar(&cmd.filters.Migrated, "migrated", false, "Get migrated instances")
	corbaCmd.Flags().BoolVar(&cmd.filters.NotMigrated, "not-migrated", false, "Get not migrated instances")
	corbaCmd.Flags().StringVarP(&cmd.filters.InstanceID, "instance-id", "i", "", "Get not migrated instances")
	corbaCmd.Flags().StringVarP(&cmd.filters.GlobalAccountID, "global-account-id", "g", "", "Filter by global account ID.")
	corbaCmd.Flags().StringVar(&cmd.source, "source", "", "File containing instances data")
	corbaCmd.Flags().IntVar(&cmd.pageStart, "pageNo", 0, "Specify which page to load")
	corbaCmd.Flags().IntVar(&cmd.pageSize, "pageSize", 5, "Specify how many elements per page to load")
	corbaCmd.Flags().IntVar(&cmd.pageLimit, "pageLimit", 2, "Specify how many pages to load, by default loads only 2 pages")

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
