package runtime

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	schema "github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// tenantHeaderName is a header key name for request send by graphQL client
const tenantHeaderName = "tenant"

// Client allows to fetch runtime's config and execute the logic against it
type Client struct {
	httpClient http.Client
	log        logrus.FieldLogger

	provisionerURL string
	instanceID     string
	tenantID       string
	kcpK8sClient   client.Client
}

func NewClient(provisionerURL, tenantID, instanceID string, clientHttp http.Client, kcpK8sClient client.Client, log logrus.FieldLogger) *Client {
	return &Client{
		tenantID:       tenantID,
		instanceID:     instanceID,
		provisionerURL: provisionerURL,
		httpClient:     clientHttp,
		log:            log,
		kcpK8sClient:   kcpK8sClient,
	}
}

type runtimeStatusResponse struct {
	Result schema.RuntimeStatus `json:"result"`
}

func (c *Client) kubeconfigForInstanceID() ([]byte, error) {
	secrets := &v1.SecretList{}
	listOpts := secretListOptions(c.instanceID)

	err := c.kcpK8sClient.List(context.Background(), secrets, listOpts...)

	if err != nil {
		return nil, fmt.Errorf("while getting secret from kcp for instanceID=%s : %w", c.instanceID, err)
	}
	if len(secrets.Items) != 1 {
		return nil, fmt.Errorf("found %d secrets for instanceID %s but expected 1", len(secrets.Items), c.instanceID)
	}
	config, ok := secrets.Items[0].Data["config"]
	if !ok {
		return nil, fmt.Errorf("while getting 'config' from secret from %s", c.instanceID)
	}
	if len(config) == 0 {
		return nil, fmt.Errorf("empty kubeconfig")
	}
	return config, nil
}

func (c *Client) FetchRuntimeConfig() (*string, error) {
	kubeconfig, err := c.kubeconfigForInstanceID()
	if err != nil {
		return nil, errors.Wrapf(err, "while getting kubeconfig %s", c.instanceID)
	}

	if err != nil {
		return nil, errors.Wrapf(err, "while getting runtime config")
	}
	if len(kubeconfig) > 0 {
		kcfg := string(kubeconfig)
		return &kcfg, nil
	}
	return nil, errors.New("kubeconfig shouldn't be nil")
}

func (c *Client) writeConfigToFile(config string) (string, error) {
	content := []byte(config)
	runtimeConfigTmpFile, err := ioutil.TempFile("", "runtime.*.yaml")
	if err != nil {
		return "", errors.Wrap(err, "while creating runtime config temp file")
	}

	if _, err := runtimeConfigTmpFile.Write(content); err != nil {
		return "", errors.Wrap(err, "while writing runtime config temp file")
	}
	if err := runtimeConfigTmpFile.Close(); err != nil {
		return "", errors.Wrap(err, "while closing runtime config temp file")
	}

	return runtimeConfigTmpFile.Name(), nil
}

func (c *Client) removeFile(fileName string) {
	err := os.Remove(fileName)
	if err != nil {
		c.log.Fatal(err)
	}
}

func (c *Client) warnOnError(err error) {
	if err != nil {
		c.log.Warn(err.Error())
	}
}

func (c *Client) objectKey(runtimeId string) client.ObjectKey {
	return client.ObjectKey{
		Namespace: "kcp-system",
		Name:      fmt.Sprintf("kubeconfig-%s", runtimeId),
	}
}

func secretListOptions(instanceID string) []client.ListOption {
	label := map[string]string{
		"kyma-project.io/instance-id": instanceID,
	}

	return []client.ListOption{
		client.InNamespace("kcp-system"),
		client.MatchingLabels(label),
	}
}
