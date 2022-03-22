package command

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"regexp"

	"github.com/pkg/errors"

	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

func NewLogsCommand() *cobra.Command {
	cmd := &LogsCommand{}

	cobraCmd := &cobra.Command{
		Use:   "logs [id]",
		Short: "Get all logs for specific regex from sc migration.",
		Args:  cobra.MaximumNArgs(1),
		PreRunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return errors.New("Missing required param `id`")
			}

			cmd.regex = args[0]
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run()
		},
	}

	cmd.corbaCmd = cobraCmd

	return cobraCmd
}

type LogsCommand struct {
	corbaCmd *cobra.Command
	regex    string
	log      logger.Logger
}

func (c *LogsCommand) Run() error {
	c.log = logger.New()

	fmt.Printf("%sLogs%s\n", Red, Reset)
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

	configOverrides := &clientcmd.ConfigOverrides{}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		c.log.Errorf("while creating kubernetes config %e", err)
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	podsList, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "while creatin a client set")
	}

	for _, pod := range podsList.Items {
		if !bytes.HasPrefix([]byte(pod.Name), []byte("base-reconciler")) {
			continue
		}

		for _, container := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
			c.printLogs(clientset, pod, container)
		}
	}

	return nil
}

func (c *LogsCommand) printLogs(clientset *kubernetes.Clientset, pod v1.Pod, container v1.Container) error {
	options := &v1.PodLogOptions{}

	options.Container = container.Name

	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, options)

	logsStream, err := req.Stream(context.TODO())

	if err != nil {
		c.log.Errorf("Error while creating a stream")
		return errors.Wrapf(err, "error in opening stream")
	}
	defer logsStream.Close()

	scanner := bufio.NewScanner(logsStream)

	searchedString := fmt.Sprintf(c.regex)
	r, err := regexp.Compile(searchedString)

	if err != nil {
		return errors.Wrapf(err, "Error during string compilation")
	}

	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		str := scanner.Text()

		positions := r.FindAllIndex([]byte(str), -1)
		occurrences := len(positions)
		if occurrences > 0 {
			c.log.Info("%s\n", str)
		}
	}

	if scanner.Err() != nil {
		return scanner.Err()
	}

	return nil
}
