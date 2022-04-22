package command

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/tools/cli/pkg/command"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers/metadata"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

func NewMetadataCommand() *cobra.Command {
	cmd := &MetadataCommand{
		metadataStorage: &metadata.Storage{},
		log:             logger.New(),
	}

	cobraCmd := &cobra.Command{
		Use:     "metadata",
		Short:   "Prints all stored metadata summary",
		Long:    "",
		Example: "",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run()
		},
	}

	cmd.corbaCmd = cobraCmd

	return cobraCmd
}

type MetadataCommand struct {
	corbaCmd        *cobra.Command
	metadataStorage *metadata.Storage
	log             logger.Logger
}

func (c *MetadataCommand) Run() error {

	reader := bufio.NewReader(os.Stdin)
	dec := json.NewDecoder(reader)

	var instances []ers.Instance
	err := dec.Decode(&instances)
	if err != nil {
		return err
	}

	fmt.Println("ID\tKEB\tmigrated\tState\tStatus\tKymaMigrated\tKymaSkipped\tProvisionedAt")
	for _, instance := range instances {
		meta, err := c.metadataStorage.Get(instance.Id)
		if err != nil {
			continue
		}
		runtime, err := c.getRuntime(instance.Id)

		fmt.Printf("%s\t%v\t%v\t%s\t%s\t%v\t%v\t%v\n", instance.Id, runtime != nil, instance.Migrated,
			instance.State, instance.Status, meta.KymaMigrated, meta.KymaSkipped, runtime.Status.CreatedAt)
	}

	return nil
}

// getRuntime gets the runtime data from KEB, returns nil, nil if not exists
func (c *MetadataCommand) getRuntime(instanceID string) (*runtime.RuntimeDTO, error) {
	fmt.Printf("%sKEB%s - ", Red, Reset)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	auth := command.CLICredentialManager(c.log)
	httpClient := oauth2.NewClient(ctx, auth)

	kebClient := runtime.NewClient(ers.GlobalOpts.KebApiUrl(), httpClient)

	runtimesPage, err := kebClient.ListRuntimes(runtime.ListParameters{InstanceIDs: []string{instanceID}})
	if err != nil {
		return nil, fmt.Errorf("while listing runtimes: %w", err)
	}

	if len(runtimesPage.Data) == 0 {
		return nil, nil
	}

	runtime := runtimesPage.Data[0]

	return &runtime, nil
}
