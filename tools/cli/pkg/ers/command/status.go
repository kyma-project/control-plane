package command

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
	mothershipAuth "github.com/kyma-project/control-plane/components/reconciler/pkg/auth"
	"github.com/kyma-project/control-plane/tools/cli/pkg/command"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers/client"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

func NewStatusCommand() *cobra.Command {
	cmd := &StatusCommand{}

	cobraCmd := &cobra.Command{
		Use:     "status [id]",
		Short:   "Check where requests sits",
		Long:    "",
		Example: "",
		Args:    cobra.MaximumNArgs(1),
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

	return cobraCmd
}

type StatusCommand struct {
	corbaCmd   *cobra.Command
	instanceID string
	log        logger.Logger
}

func (c *StatusCommand) Run() error {
	c.log = logger.New()

	// ers
	fmt.Printf("%sERS%s - ", Red, Reset)
	ersClient, err := client.NewErsClient()
	if err != nil {
		return fmt.Errorf("while initializing ers client: %w", err)
	}
	defer ersClient.Close()

	instance, err := ersClient.GetOne(c.instanceID)
	if err != nil {
		return fmt.Errorf("while getting ers instance: %w", err)
	}

	jsonData, err := ToJson(instance)
	if err != nil {
		return fmt.Errorf("while creating a json: %w", err)
	}
	fmt.Printf("%s - %s\n", instance.State, instance.StateMessage)
	fmt.Printf("%s - %s\n", "Status", instance.Status)
	fmt.Printf("Migrated - %t\n", instance.Migrated)
	c.log.Debugf(jsonData)

	// KEB
	fmt.Printf("%sKEB%s - ", Red, Reset)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	auth := command.CLICredentialManager(c.log)
	httpClient := oauth2.NewClient(ctx, auth)

	kebClient := runtime.NewClient(ers.GlobalOpts.KebApiUrl(), httpClient)

	runtimesPage, err := kebClient.ListRuntimes(runtime.ListParameters{InstanceIDs: []string{c.instanceID}})
	if err != nil {
		return fmt.Errorf("while listing runtimes: %w", err)
	}

	if len(runtimesPage.Data) != 1 {
		return fmt.Errorf("No data in KEB")
	}

	runtime := runtimesPage.Data[0]
	jsonData, err = ToJson(runtime)
	if err != nil {
		return fmt.Errorf("while creating a json: %w", err)
	}
	fmt.Printf("%s\n", runtime.Status.State)
	fmt.Printf("Provisioning: %s %s\n", runtime.Status.Provisioning.State, runtime.Status.Provisioning.CreatedAt)

	if runtime.Status.UpgradingCluster != nil {
		for _, v := range runtime.Status.UpgradingCluster.Data {
			fmt.Printf("Upgrade Cluster: %s %s\n", v.State, v.CreatedAt)
		}
	}

	if runtime.Status.UpgradingKyma != nil {
		for _, v := range runtime.Status.UpgradingKyma.Data {
			fmt.Printf("Upgrade Kyma: %s %s\n", v.State, v.CreatedAt)
		}
	}

	if runtime.Status.Update != nil {
		for _, v := range runtime.Status.Update.Data {
			fmt.Printf("Update Kyma: %s %s\n", v.State, v.CreatedAt)
		}
	}

	if runtime.Status.Suspension != nil {
		for _, v := range runtime.Status.Suspension.Data {
			fmt.Printf("Suspensions: %s %s\n", v.State, v.CreatedAt)
		}
	}

	if runtime.Status.Unsuspension != nil {
		for _, v := range runtime.Status.Unsuspension.Data {
			fmt.Printf("Unsuspension: %s %s\n", v.State, v.CreatedAt)
		}
	}

	if runtime.Status.Deprovisioning != nil {
		fmt.Printf("Deprovisiong: %s %s\n", runtime.Status.Deprovisioning.State, runtime.Status.Deprovisioning.CreatedAt)
	}
	c.log.Debugf("%s", jsonData)

	// Mothership
	fmt.Printf("%sMOTHERSHIP%s", Red, Reset)
	mothershipClient, err := mothershipAuth.NewClient(ers.GlobalOpts.MothershipApiUrl(), httpClient)
	if err != nil {
		return fmt.Errorf("while creating mothership client: %w", err)
	}

	response, err := mothershipClient.GetClustersState(ctx, &mothership.GetClustersStateParams{
		RuntimeID: &runtime.RuntimeID,
	})
	if err != nil {
		return fmt.Errorf("wile getting cluster state: %w", err)
	}

	defer response.Body.Close()

	if IsErrResponse(response.StatusCode) {
		err := ResponseErr(response)
		return err
	}

	var result mothership.HTTPClusterStateResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("Error getting %w", command.ErrMothershipResponse)
	}

	result.Configuration.Components = nil
	jsonData, err = ToJson(result)
	if err != nil {
		return fmt.Errorf("while creating a json %w: ", err)
	}

	fmt.Printf("- %s\n last reconcilled: %s\n deleted: %v\n", *result.Status.Status, result.Status.Created, *result.Status.Deleted)
	c.log.Debugf(jsonData)

	// logs
	fmt.Printf("%sLogs%s\n", Red, Reset)
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// if you want to change the loading rules (which files in which order), you can do so here

	configOverrides := &clientcmd.ConfigOverrides{}
	// if you want to change override values or bind them to flags, there are methods to help you

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		c.log.Errorf("while creating kubernetes config %e", err)
		return err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	podsList, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "component=sc-migration",
	})
	if err != nil {
		return err // todo: logs
	}

	for _, pod := range podsList.Items {
		for _, container := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
			options := &corev1.PodLogOptions{}

			// options.Follow = true
			// options.TailLines = &[]int64{int64(100)}[0]
			options.Container = container.Name

			req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, options)

			logsStream, err := req.Stream(context.Background())
			if err != nil {
				return fmt.Errorf("error in opening stream: %w", err)
			}

			scanner := bufio.NewScanner(logsStream)

			searchedString := fmt.Sprintf(".*%s.*", runtime.RuntimeID)
			r, _ := regexp.Compile(searchedString)

			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				str := scanner.Text()

				positions := r.FindAllIndex([]byte(str), -1)
				occurrences := len(positions)
				if occurrences > 0 {
					fmt.Printf("%s\n", str)

				}
			}

			if scanner.Err() != nil {
				return scanner.Err()
			}

			err = logsStream.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
