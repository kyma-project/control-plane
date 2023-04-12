package skrlisteners

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"testing"
	"time"
)

type TestInstances struct {
	testEnvs  []*envtest.Environment
	instances []internal.Instance
	db        storage.BrokerStorage
}

func Test(t *testing.T) {
	test := TestInstances{}
	test.db = storage.NewMemoryStorage()
	test.PrepareFakeData()
	t.Run("", func(t *testing.T) {
		fmt.Print("a")
	})
}

func (t *TestInstances) PrepareFakeData() {
	defer t.clean()
	for i := 0; i < 10; i++ {
		t.PrepareData()
	}
}

func (t *TestInstances) clean() {
	for _, te := range t.testEnvs {
		te.Stop()
	}
}

func (t *TestInstances) PrepareData() {
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		return
	}
	t.testEnvs = append(t.testEnvs, testEnv)
	i := t.SpamInstance(cfg.String())
	t.instances = append(t.instances, *i)
	err = t.db.Instances().Insert(*i)
	if err != nil {

	}
}

func (t *TestInstances) SpamInstance(kubeConfig string) *internal.Instance {
	return &internal.Instance{
		InstanceID:                  "",
		RuntimeID:                   "",
		GlobalAccountID:             "",
		SubscriptionGlobalAccountID: "",
		SubAccountID:                "",
		ServiceID:                   "",
		ServiceName:                 "",
		ServicePlanID:               "",
		ServicePlanName:             "",
		DashboardURL:                "",
		Parameters: internal.ProvisioningParameters{
			PlanID:     "",
			ServiceID:  "",
			ErsContext: internal.ERSContext{},
			Parameters: internal.ProvisioningParametersDTO{
				AutoScalerParameters:        internal.AutoScalerParameters{},
				Name:                        "",
				TargetSecret:                nil,
				VolumeSizeGb:                nil,
				MachineType:                 nil,
				Region:                      nil,
				Purpose:                     nil,
				LicenceType:                 nil,
				Zones:                       nil,
				OptionalComponentsToInstall: nil,
				KymaVersion:                 "",
				OverridesVersion:            "",
				RuntimeAdministrators:       nil,
				Provider:                    nil,
				Kubeconfig:                  kubeConfig,
				ShootName:                   "",
				ShootDomain:                 "",
				OIDC:                        nil,
			},
			PlatformRegion:   "",
			PlatformProvider: "",
		},
		ProviderRegion:  "",
		InstanceDetails: internal.InstanceDetails{},
		CreatedAt:       time.Time{},
		UpdatedAt:       time.Time{},
		DeletedAt:       time.Time{},
		ExpiredAt:       nil,
		Version:         0,
		Provider:        "",
	}
}
