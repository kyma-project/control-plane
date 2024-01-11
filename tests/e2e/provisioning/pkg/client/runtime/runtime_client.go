package runtime

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/director"
	schema "github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// tenantHeaderName is a header key name for request send by graphQL client
const tenantHeaderName = "tenant"

// Client allows to fetch runtime's config and execute the logic against it
type Client struct {
	httpClient     http.Client
	directorClient *director.Client
	log            logrus.FieldLogger

	provisionerURL string
	instanceID     string
	tenantID       string
	kcpK8sClient   client.Client
}

func NewClient(provisionerURL, tenantID, instanceID string, clientHttp http.Client, directorClient *director.Client, kcpK8sClient client.Client, log logrus.FieldLogger) *Client {
	return &Client{
		tenantID:       tenantID,
		instanceID:     instanceID,
		provisionerURL: provisionerURL,
		httpClient:     clientHttp,
		directorClient: directorClient,
		log:            log,
		kcpK8sClient:   kcpK8sClient,
	}
}

type runtimeStatusResponse struct {
	Result schema.RuntimeStatus `json:"result"`
}

type SecretProvider struct {
	kcpK8sClient client.Client
}

func (c *Client) kubeconfigForRuntimeID(runtimeId string) ([]byte, error) {
	kubeConfigSecret := &v1.Secret{}
	err := c.kcpK8sClient.Get(context.Background(), c.objectKey(runtimeId), kubeConfigSecret)
	if err != nil {
		return nil, fmt.Errorf("while getting secret from kcp for runtimeId=%s : %w", runtimeId, err)
	}
	config, ok := kubeConfigSecret.Data["config"]
	if !ok {
		return nil, fmt.Errorf("while getting 'config' from secret from %s", c.objectKey(runtimeId))
	}
	if len(config) == 0 {
		return nil, fmt.Errorf("empty kubeconfig")
	}
	return config, nil
}

func (c *Client) FetchRuntimeConfig() (*string, error) {
	runtimeID, err := c.directorClient.GetRuntimeID(c.tenantID, c.instanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "while getting runtime id from director for instance ID %s", c.instanceID)
	}

	kubeconfig, err := c.kubeconfigForRuntimeID(runtimeID)
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
