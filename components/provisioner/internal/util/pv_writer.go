package util

import (
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"os"
	"sigs.k8s.io/yaml"
)

const pvMountPath = "/testdata/provisioner"

func WriteToPV(shoot *v1beta1.Shoot) error {

	file, err := os.OpenFile(pvMountPath+"/"+shoot.Name+"-gardenerShoot.yaml", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// Remove managed fields from the object
	shoot.ManagedFields = nil
	b, err := yaml.Marshal(shoot)
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to marshal Shoot object: %w", err)
	}

	_, err = file.Write(b)
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to write to file: %w", err)
	}

	if err := file.Close(); err != nil {
		return err
	}

	return nil
}
