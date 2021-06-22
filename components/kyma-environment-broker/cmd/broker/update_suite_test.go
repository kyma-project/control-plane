package main

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	kebRuntime "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type UpdateSuite struct {
	HttpSuite
}

func (s *UpdateSuite) TearDown() {
	s.httpServer.Close()
}

func NewUpdateSuite(t *testing.T) *UpdateSuite {
	cfg := fixConfig()

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
	}, defaultKymaVer, map[string]string{"cf-eu10": "europe"}, cfg.FreemiumProviders)

	db := storage.NewMemoryStorage()

	require.NoError(t, err)

	logs := logrus.New()

	httpSuite := NewHttpSuite(t)
	httpSuite.CreateAPI(inputFactory, cfg, db, nil, nil, logs)

	return &UpdateSuite{
		HttpSuite: httpSuite,
	}
}
