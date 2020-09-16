package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	gardenerapi "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerFake "github.com/gardener/gardener/pkg/client/core/clientset/versioned/fake"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_kyma"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	kebRuntime "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	globalAccountLabel  = "account"
	subAccountLabel     = "subaccount"
	runtimeIDAnnotation = "kcp.provisioner.kyma-project.io/runtime-id"
	platformRegion      = "cf-eu10"
)

type OrchestrationSuite struct {
	gardenerNamespace  string
	provisionerClient  *provisioner.FakeClient
	orchestrationQueue *process.Queue
	storage            storage.BrokerStorage
	gardenerClient     *gardenerFake.Clientset

	t *testing.T
}

func NewOrchestrationSuite(t *testing.T) *OrchestrationSuite {
	logs := logrus.New()
	logs.Formatter.(*logrus.TextFormatter).TimestampFormat = "15:04:05.000"
	optionalComponentsDisablers := kebRuntime.ComponentsDisablers{}
	optComponentsSvc := kebRuntime.NewOptionalComponentsService(optionalComponentsDisablers)

	disabledComponentsProvider := kebRuntime.NewDisabledComponentsProvider()

	componentListProvider := &automock.ComponentListProvider{}
	componentListProvider.On("AllComponents", mock.Anything).Return([]v1alpha1.KymaComponent{}, nil)

	inputFactory, err := input.NewInputBuilderFactory(optComponentsSvc, disabledComponentsProvider, componentListProvider, input.Config{
		MachineImageVersion:         "coreos",
		KubernetesVersion:           "1.18",
		MachineImage:                "253",
		Timeout:                     time.Minute,
		URL:                         "http://localhost",
		DefaultGardenerShootPurpose: "testing",
	}, "1.15.1", map[string]string{"cf-eu10": "europe"})
	require.NoError(t, err)

	ctx, _ := context.WithTimeout(context.Background(), 20*time.Minute)
	db := storage.NewMemoryStorage()
	sch := runtime.NewScheme()
	require.NoError(t, coreV1.AddToScheme(sch))
	cli := fake.NewFakeClientWithScheme(sch)

	gardenerClient := gardenerFake.NewSimpleClientset()
	provisionerClient := provisioner.NewFakeClient()
	const gardenerProject = "testing"
	gardenerNamespace := fmt.Sprintf("garden-%s", gardenerProject)

	eventBroker := event.NewPubSub()

	kymaQueue, err := NewOrchestrationProcessingQueue(ctx, db, cli, provisionerClient, gardenerClient.CoreV1beta1(),
		gardenerNamespace, eventBroker, inputFactory, &upgrade_kyma.IntervalConfig{
			Retry:              10 * time.Millisecond,
			StatusCheck:        100 * time.Millisecond,
			UpgradeKymaTimeout: 2 * time.Second,
		}, 250*time.Millisecond, logs)

	return &OrchestrationSuite{
		gardenerNamespace:  gardenerNamespace,
		provisionerClient:  provisionerClient,
		orchestrationQueue: kymaQueue,
		storage:            db,
		gardenerClient:     gardenerClient,

		t: t,
	}
}

type RuntimeOptions struct {
	GlobalAccountID string
	SubAccountID    string
	PlatformRegion  string
	Region          string
}

func (o *RuntimeOptions) ProvideRegion() *string {
	if o.Region != "" {
		return &o.Region
	} else {
		r := "westeurope"
		return &r
	}
}

func (o *RuntimeOptions) ProvideGlobalAccountID() string {
	if o.GlobalAccountID != "" {
		return o.GlobalAccountID
	} else {
		return uuid.New()
	}
}

func (o *RuntimeOptions) ProvideSubAccountID() string {
	if o.SubAccountID != "" {
		return o.SubAccountID
	} else {
		return uuid.New()
	}
}

func (o *RuntimeOptions) ProvidePlatformRegion() string {
	if o.PlatformRegion != "" {
		return o.PlatformRegion
	} else {
		return "cf-eu10"
	}
}

func (s *OrchestrationSuite) CreateProvisionedRuntime(options RuntimeOptions) string {
	planID := broker.AzurePlanID
	runtimeID := uuid.New()
	globalAccountID := options.ProvideGlobalAccountID()
	subAccountID := options.ProvideSubAccountID()
	instanceID := uuid.New()

	instance := internal.Instance{
		RuntimeID:     runtimeID,
		ServicePlanID: planID,
		InstanceID:    instanceID,
	}
	provisioningParameters := internal.ProvisioningParameters{
		PlanID: planID,
		ErsContext: internal.ERSContext{
			GlobalAccountID: globalAccountID,
			SubAccountID:    subAccountID,
		},
		PlatformRegion: options.ProvidePlatformRegion(),
		Parameters: internal.ProvisioningParametersDTO{
			Region: options.ProvideRegion(),
		},
	}
	serializedProvisioningParams, err := json.Marshal(provisioningParameters)
	require.NoError(s.t, err)
	provisioningOperation := internal.ProvisioningOperation{
		Operation: internal.Operation{
			State:      internal.Succeeded,
			ID:         uuid.New(),
			InstanceID: instanceID,
		},
		RuntimeID:              instance.RuntimeID,
		ProvisioningParameters: string(serializedProvisioningParams),
	}
	shoot := &gardenerapi.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("shoot%s", runtimeID),
			Namespace: s.gardenerNamespace,
			Labels: map[string]string{
				globalAccountLabel: globalAccountID,
				subAccountLabel:    subAccountID,
			},
			Annotations: map[string]string{
				runtimeIDAnnotation: runtimeID,
			},
		},
		Spec: gardenerapi.ShootSpec{
			Region: *options.ProvideRegion(),
			Maintenance: &gardenerapi.Maintenance{
				TimeWindow: &gardenerapi.MaintenanceTimeWindow{
					Begin: "030000+0000",
					End:   "040000+0000",
				},
			},
		},
	}

	s.storage.Instances().Insert(instance)
	s.storage.Operations().InsertProvisioningOperation(provisioningOperation)
	s.gardenerClient.CoreV1beta1().Shoots(s.gardenerNamespace).Create(shoot)

	return runtimeID
}

func (s *OrchestrationSuite) CreateOrchestration(runtimeID string) string {
	params, err := json.Marshal(orchestration.Parameters{
		Targets: internal.TargetSpec{
			Include: []internal.RuntimeTarget{
				{RuntimeID: runtimeID},
			},
		},
	})
	require.NoError(s.t, err)
	now := time.Now()
	o := internal.Orchestration{
		OrchestrationID: uuid.New(),
		State:           internal.Pending,
		Description:     "started processing of Kyma upgrade",
		Parameters: sql.NullString{
			String: string(params),
			Valid:  true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.storage.Orchestrations().Insert(o)

	s.orchestrationQueue.Add(o.OrchestrationID)
	return o.OrchestrationID
}

func (s *OrchestrationSuite) FinishUpgradeOperationByProvisioner(runtimeID string) {
	status := s.provisionerClient.FindOperationByRuntimeIDAndType(runtimeID, gqlschema.OperationTypeUpgrade)
	if status.ID != nil {
		s.provisionerClient.FinishProvisionerOperation(*status.ID)
	}
}

func (s *OrchestrationSuite) WaitForOrchestrationState(orchestrationID string, state string) {
	var orchestration *internal.Orchestration
	err := wait.PollImmediate(100*time.Millisecond, 15*time.Second, func() (done bool, err error) {
		orchestration, _ = s.storage.Orchestrations().GetByID(orchestrationID)
		return orchestration.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the orchestration expected state %s. The existing orchestration %+V", state, orchestration)
}
