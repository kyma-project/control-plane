package testkit

import (
	"encoding/json"
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"io"
	"os"
	"path"
	"sigs.k8s.io/yaml"
)

// TestDataWriter can be used for dumping test data such as shoot specs, and Graphql mutations to files
// It was introduced to support migration to Kyma Infrastructure Manager (https://github.com/kyma-project/infrastructure-manager/issues/185)
type TestDataWriter struct {
	shootNamespace string
	directory      string
	enabled        bool
}

func NewTestDataWriter(shootNamespace, directory string, enabled bool) *TestDataWriter {
	return &TestDataWriter{
		shootNamespace: shootNamespace,
		directory:      directory,
		enabled:        enabled,
	}
}

func (tdw *TestDataWriter) PersistShoot(shoot *v1beta1.Shoot) (string, error) {
	fileName := fmt.Sprintf("%s-%s-shootCR.yaml", shoot.Name, tdw.shootNamespace)
	filePath := path.Join(tdw.directory, fileName)

	writer, err := getWriter(filePath)
	if err != nil {
		return "", fmt.Errorf("unable to create file: %w", err)
	}

	b, err := yaml.Marshal(shoot)
	if err != nil {
		return "", fmt.Errorf("unable to marshal shoot: %w", err)
	}

	if _, err = writer.Write(b); err != nil {
		return "", fmt.Errorf("unable to write to file: %w", err)
	}
	return filePath, nil
}

func (tdw *TestDataWriter) PersistGraphQL(mutation gqlschema.ProvisionRuntimeInput) (string, error) {
	fileName := fmt.Sprintf("%s-%s-mutation.json", mutation.ClusterConfig.GardenerConfig.Name, tdw.shootNamespace)
	filePath := path.Join(tdw.directory, fileName)

	writer, err := getWriter(filePath)
	if err != nil {
		return "", fmt.Errorf("unable to create file: %w", err)
	}

	b, err := json.Marshal(mutation)
	if err != nil {
		return "", fmt.Errorf("unable to marshal GraphQL mutation: %w", err)
	}

	if _, err = writer.Write(b); err != nil {
		return "", fmt.Errorf("unable to write to file: %w", err)
	}
	return filePath, nil
}

func getWriter(filePath string) (io.Writer, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to create file: %w", err)
	}
	return file, nil
}

func (tdw *TestDataWriter) Enabled() bool {
	return tdw.enabled
}
