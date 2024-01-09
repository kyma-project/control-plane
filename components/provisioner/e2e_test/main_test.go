package main_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/machinebox/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/vrischmann/envconfig"
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

type OperationStatusResp struct {
	OperationStatus struct {
		Operation string `json:"operation"`
		State     string `json:"state"`
	} `json:"runtimeOperationStatus"`
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

	req := graphql.NewRequest(string(payload))

	for key, val := range vars {
		req.Var(key, val)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Tenant", tenant)

	err = gql.client.Run(ctx, req, resp)

	return err
}

func (gql GQLClient) provision(
	ctx context.Context,
	name,
	provider,
	provider_secret,
	kubernetes_version,
	machine_type,
	disk_type,
	region string,
) (resp ProvisionResp, err error) {
	err = gql.gqlRequest(
		ctx,
		"provision.graphql",
		map[string]string{
			"name":              name,
			"provider":          provider,
			"providerSecret":    provider_secret,
			"kubernetesVersion": kubernetes_version,
			"machineType":       machine_type,
			"diskType":          disk_type,
			"region":            region,
		},
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

func (gql GQLClient) runtimeStatus(ctx context.Context, runtimeID string) (resp StatusResp, err error) {
	err = gql.gqlRequest(
		ctx,
		"status.graphql",
		map[string]string{"runtimeID": runtimeID},
		&resp)
	return
}

func (gql GQLClient) operationStatus(ctx context.Context, operationID string) (resp OperationStatusResp, err error) {
	fmt.Println("Getting operation status for ", operationID)

	err = gql.gqlRequest(
		ctx,
		"operationstatus.graphql",
		map[string]string{"operationID": operationID},
		&resp)
	return
}

func (gql GQLClient) waitForOp(ctx context.Context, operationID string) (resp OperationStatusResp, err error) {
	start := time.Now()
	defer func() {
		fmt.Printf("done in: %s", time.Since(start))
	}()
	for {
		resp, err = gql.operationStatus(ctx, operationID)
		if err != nil {
			return
		}

		fmt.Printf("resp:[%s:%s]\n", resp.OperationStatus.Operation, resp.OperationStatus.State)

		if resp.OperationStatus.State == "InProgress" {
			if time.Since(start) > waitTimeout {
				return
			}
			time.Sleep(waitDelay)
			continue
		}

		return
	}
}

type testConfig struct {
	ProviderSecret    string `envconfig:"GARDENER_SECRET_NAME"`
	Provider          string `envconfig:"GARDENER_PROVIDER,default=gcp"`
	KubernetesVersion string `envconfig:"default=1.26.11"`
	MachineType       string `envconfig:"default=e2-medium"`
	DiskType          string `envconfig:"default=pd-balanced"`
	Region            string `envconfig:"default=europe-west3"`
}

func TestName(t *testing.T) {
	if os.Getenv("APP_GARDENER_KUBECONFIG_PATH") == "" {
		t.SkipNow()
	}

	var conf testConfig
	if err := envconfig.Init(&conf); err != nil {
		panic(err)
	}

	ctx := context.Background()

	cli := GQLClient{
		reqsPath:       "./requests/",
		providerSecret: conf.ProviderSecret,
		provider:       conf.Provider,
		client:         graphql.NewClient("http://localhost:3000/graphql"),
	}

	name := fmt.Sprintf("pts%d", time.Now().Unix()%1000000)

	t.Logf("Provisioning a %s cluster - %s", conf.Provider, name)
	provisionResp, err := cli.provision(
		ctx,
		name,
		conf.Provider,
		conf.ProviderSecret,
		conf.KubernetesVersion,
		conf.MachineType,
		conf.DiskType,
		conf.Region,
	)
	assert.NoError(t, err)
	t.Log(provisionResp)

	operationID := provisionResp.ProvisionRuntime.Id
	runtimeID := provisionResp.ProvisionRuntime.RuntimeID

	t.Logf("Waiting for %s to provision, runtimeID %s, OperationID %s", name, runtimeID, operationID)
	statusResp, err := cli.waitForOp(ctx, operationID)
	assert.NoError(t, err)
	t.Log(statusResp)

	t.Logf("Deprovisioning cluster %s", name)
	deprovisionResp, err := cli.deprovision(ctx, runtimeID)
	assert.NoError(t, err)
	t.Log(deprovisionResp)
}
