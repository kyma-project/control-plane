package main_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/machinebox/graphql"
	"github.com/stretchr/testify/assert"
)

const tenant = "3e64ebae-38b5-46a0-b1ed-9ccee153a0ae"
const waitDelay = 30 * time.Second
const waitTimeout = 40 * time.Minute

type ProvisionResp struct {
	ProvisionRuntime struct {
		Id        string `json:"id"`
		Operation string `json:"operation"`
		State     string `json:"state"`
		Message   string `json:"message"`
		RuntimeID string `json:"runtimeID"`
	} `json:"provisionRuntime"`
}

type StatusResp struct {
	RuntimeStatus struct {
		RuntimeConfiguration struct {
			ClusterConfig struct {
				VolumeSizeGB int `json:"volumeSizeGB"`
			} `json:"clusterConfig"`
		} `json:"runtimeConfiguration"`
		LastOperationStatus struct {
			Operation string `json:"operation"`
			Message   string `json:"message"`
			LastError struct {
				ErrMessage string `json:"errMessage"`
			} `json:"lastError"`
		} `json:"lastOperationStatus"`
	} `json:"runtimeStatus"`
}

type DeprovisionResp struct {
	DeprovisionRuntime string `json:"deprovisionRuntime"`
}

type GQLClient struct {
	reqsPath       string
	providerSecret string
	provider       string
	client         *graphql.Client
}

func (gql GQLClient) gqlRequest(ctx context.Context, reqName string, vars map[string]string, resp interface{}) error {
	payload, err := os.ReadFile(path.Join(gql.reqsPath, reqName))
	if err != nil {
		return err
	}

	//$name: String!, $provider: String!, $provider_secret: String
	req := graphql.NewRequest(string(payload))

	for key, val := range vars {
		req.Var(key, val)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Tenant", tenant)

	err = gql.client.Run(ctx, req, resp)

	return err
}

func (gql GQLClient) provision(ctx context.Context, name, provider, provider_secret string) (resp ProvisionResp, err error) {
	err = gql.gqlRequest(
		ctx,
		"provision.graphql",
		map[string]string{"name": name, "provider": provider, "provider_secret": provider_secret},
		&resp)
	return
}

func (gql GQLClient) deprovision(ctx context.Context, runtimeID string) (resp DeprovisionResp, err error) {
	err = gql.gqlRequest(
		ctx,
		"deprovision.graphql",
		map[string]string{"runtimeID": runtimeID},
		&resp)
	return
}

func (gql GQLClient) status(ctx context.Context, runtimeID string) (resp StatusResp, err error) {
	err = gql.gqlRequest(
		ctx,
		"status.graphql",
		map[string]string{"runtimeID": runtimeID},
		&resp)
	return
}

func (gql GQLClient) waitForOp(ctx context.Context, runtimeID string) (resp StatusResp, err error) {
	start := time.Now()
	defer fmt.Println()
	for {
		resp, err = gql.status(ctx, runtimeID)
		if err != nil {
			return
		}

		msg := resp.RuntimeStatus.LastOperationStatus.Message

		if strings.HasPrefix(msg, "Operation in progress.") ||
			strings.HasSuffix(msg, "started") {
			if time.Since(start) > waitTimeout {
				return
			}
			fmt.Print(".")
			time.Sleep(waitDelay)
			continue
		}

		return
	}
}

func TestName(t *testing.T) {
	if os.Getenv("APP_GARDENER_KUBECONFIG_PATH") == "" {
		t.SkipNow()
	}

	ctx := context.Background()

	providerSecret := os.Getenv("GARDENER_SECRET_NAME")
	provider := os.Getenv("GARDENER_PROVIDER")
	cli := GQLClient{
		reqsPath:       "./requests/",
		providerSecret: providerSecret,
		provider:       provider,
		client:         graphql.NewClient("http://localhost:3000/graphql"),
	}

	name := fmt.Sprintf("pts%d", time.Now().Unix()%1000000)

	t.Logf("Provisioning a %s cluster - %s", provider, name)
	provisionResp, err := cli.provision(ctx, name, provider, providerSecret)
	assert.NoError(t, err)
	t.Log(provisionResp)

	runtimeID := provisionResp.ProvisionRuntime.RuntimeID

	t.Logf("Waiting for %s to provision", name)
	statusResp, err := cli.waitForOp(ctx, runtimeID)
	assert.NoError(t, err)
	t.Log(statusResp)

	t.Logf("Deprovisioning cluster %s", name)
	deprovisionResp, err := cli.deprovision(ctx, runtimeID)
	assert.NoError(t, err)
	t.Log(deprovisionResp)
}
