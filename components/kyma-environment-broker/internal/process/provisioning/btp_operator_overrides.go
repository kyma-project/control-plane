package provisioning

import (
	"encoding/json"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

const BTPOperatorComponentName = "sap-btp-operator"

type BTPOperatorOverridesStep struct {
	operationManager *process.ProvisionOperationManager
}

type creds struct {
	ClientID          string `json:"clientid"`
	ClientSecret      string `json:"clientsecret"`
	ServiceManagerURL string `json:"sm_url"`
	URL               string `json:"url"`
}

func NewBTPOperatorOverridesStep(os storage.Operations) *BTPOperatorOverridesStep {
	return &BTPOperatorOverridesStep{
		operationManager: process.NewProvisionOperationManager(os),
	}
}

func (s *BTPOperatorOverridesStep) Name() string {
	return "BTPOperatorOverrides"
}

func (s *BTPOperatorOverridesStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return operation, 0, err
	}
	resp, err := smCli.GetBinding(operation.BTPOperator.Instance.InstanceKey(), operation.BTPOperator.BindingID)
	if err != nil {
		return operation, 0, err
	}
	creds := creds{}
	if err := json.Unmarshal(resp.Credentials, &creds); err != nil {
		return operation, 0, err
	}

	overrides := []*gqlschema.ConfigEntryInput{
		{
			Key:    "manager.secret.clientid",
			Value:  creds.ClientID,
			Secret: ptr.Bool(true),
		},
		{
			Key:    "manager.secret.clientsecret",
			Value:  creds.ClientSecret,
			Secret: ptr.Bool(true),
		},
		{
			Key:   "manager.secret.url",
			Value: creds.ServiceManagerURL,
		},
		{
			Key:   "manager.secret.tokenurl",
			Value: creds.URL,
		},
		// TODO: figure out where to get cluster ID
		/*
			{
				Key:   "cluster.id",
				Value: smctl.clusterid,
			},
		*/
	}
	operation.InputCreator.AppendOverrides(BTPOperatorComponentName, overrides)
	return operation, 0, nil
}
