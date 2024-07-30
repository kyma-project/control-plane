package util

import (
	"encoding/json"
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"io"
	"os"
	"sigs.k8s.io/yaml"
)

func PersistShoot(path string, shoot *v1beta1.Shoot) error {
	writer, err := getWriter(path)
	if err != nil {
		return fmt.Errorf("unable to create file: %w", err)
	}

	b, err := yaml.Marshal(shoot)
	if err != nil {
		return fmt.Errorf("unable to marshal shoot: %w", err)
	}

	if _, err = writer.Write(b); err != nil {
		return fmt.Errorf("unable to write to file: %w", err)
	}
	return nil
}

func getWriter(filePath string) (io.Writer, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to create file: %w", err)
	}
	return file, nil
}

func PersistGraphQL(path string, mutation gqlschema.ProvisionRuntimeInput) error {
	writer, err := getWriter(path)
	if err != nil {
		return fmt.Errorf("unable to create file: %w", err)
	}

	b, err := json.Marshal(mutation)
	if err != nil {
		return fmt.Errorf("unable to marshal GraphQL mutation: %w", err)
	}

	if _, err = writer.Write(b); err != nil {
		return fmt.Errorf("unable to write to file: %w", err)
	}
	return nil
}
