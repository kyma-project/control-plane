package command

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
	mothershipAuth "github.com/kyma-project/control-plane/components/reconciler/pkg/auth"
	"github.com/kyma-project/control-plane/tools/cli/pkg/command"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func NewPreviewsCommand() *cobra.Command {
	cmd := &PreviewsCommand{}

	cobraCmd := &cobra.Command{
		Use:     "previews [id]",
		Short:   "Check where requests sits",
		Long:    "",
		Example: "",
		Args:    cobra.MaximumNArgs(1),
		PreRunE: func(_ *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run()
		},
	}

	cobraCmd.Flags().BoolVarP(&cmd.deprovision, "deprovision", "d", false, "Get not migrated instances")

	cmd.corbaCmd = cobraCmd

	return cobraCmd
}

type PreviewsCommand struct {
	corbaCmd    *cobra.Command
	log         logger.Logger
	deprovision bool
}

func (c *PreviewsCommand) Run() error {
	c.log = logger.New()

	// ers
	// fmt.Printf("%sERS%s - ", Red, Reset)
	// ersClient, err := client.NewErsClient()
	// if err != nil {
	// 	return fmt.Errorf("while initializing ers client: %w", err)
	// }
	// defer ersClient.Close()

	// instance, err := ersClient.GetOne(c.instanceID)
	// if err != nil {
	// 	return fmt.Errorf("while getting ers instance: %w", err)
	// }

	// jsonData, err := ToJson(instance)
	// if err != nil {
	// 	return fmt.Errorf("while creating a json: %w", err)
	// }
	// fmt.Printf("%s - %s\n", instance.State, instance.StateMessage)
	// fmt.Printf("Migrated - %t\n", instance.Migrated)
	// c.log.Debugf(jsonData)

	// KEB
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	auth := command.CLICredentialManager(c.log)
	httpClient := oauth2.NewClient(ctx, auth)

	kebClient := runtime.NewClient(ers.GlobalOpts.KebApiUrl(), httpClient)

	// Mothership Client
	mothershipClient, err := mothershipAuth.NewClient(ers.GlobalOpts.MothershipApiUrl(), httpClient)
	if err != nil {
		return fmt.Errorf("while creating mothership client: %w", err)
	}

	runtimesPage, err := kebClient.ListRuntimes(runtime.ListParameters{Plans: []string{"preview"}})
	if err != nil {
		return fmt.Errorf("while listing runtimes: %w", err)
	}

	if len(runtimesPage.Data) == 0 {
		return fmt.Errorf("no data in KEB")
	}

	for _, runtime := range runtimesPage.Data {

		// Mothership
		response, err := mothershipClient.GetClustersState(ctx, &mothership.GetClustersStateParams{
			RuntimeID: &runtime.RuntimeID,
		})
		if err != nil {
			return fmt.Errorf("wile getting cluster state: %w", err)
		}

		defer response.Body.Close()

		if IsErrResponse(response.StatusCode) {
			err := ResponseErr(response)
			fmt.Errorf("error getting %w", err)
			continue
		}

		var result mothership.HTTPClusterStateResponse
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			return fmt.Errorf("error getting %w", command.ErrMothershipResponse)
		}

		result.Configuration.Components = nil
		if err != nil {
			return fmt.Errorf("while creating a json %w: ", err)
		}

		fmt.Printf("Provisioning of global account %s, instance id %s and ", runtime.GlobalAccountID, runtime.InstanceID)
		if runtime.RuntimeID == "" {
			runtime.RuntimeID = "[Not Found]"
		}

		fmt.Printf(" runtime Id %s state: %s - ", runtime.RuntimeID, runtime.Status.Provisioning.State)
		fmt.Printf("Reconciler: %s\n", *result.Status.Status)

		if c.deprovision {
			fmt.Printf("Removing")
			kebClient.Deprovision(runtime.InstanceID)
		}
	}

	return nil
}
