package installation

import (
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/config"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/deployment"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/overrides"

	"github.com/pkg/errors"
)

type Deployer struct {
	RuntimeID string
	Cfg       *config.Config
	Builder   *overrides.Builder
	Callback  func(string) func(deployment.ProcessUpdate)
}

func (d *Deployer) StartKymaDeployment() error {
	deployer, err := deployment.NewDeployment(d.Cfg, d.Builder, d.Callback(d.RuntimeID))
	if err != nil {
		return errors.Wrap(err, "while creating deployer")
	}

	err = deployer.StartKymaDeployment()
	if err != nil {
		return errors.Wrap(err, "while starting Kyma deployment")
	}

	return nil
}
