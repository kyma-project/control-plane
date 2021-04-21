package broker

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

func TestSchemaGenerator(t *testing.T) {
	tests := []struct {
		name         string
		generator    func([]string) []byte
		machineTypes []string
		file         string
	}{
		{
			name:         "AWS schema is correct",
			generator:    AWSSchema,
			machineTypes: []string{"m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge", "m4.2xlarge", "m4.4xlarge", "m4.10xlarge", "m4.16xlarge"},
			file:         "aws-schema.json",
		},
		{
			name:         "AWS HA schema is correct",
			generator:    AWSHASchema,
			machineTypes: []string{"m5d.xlarge"},
			file:         "aws-ha-schema.json",
		},
		{
			name:         "Azure schema is correct",
			generator:    AzureSchema,
			machineTypes: []string{"Standard_D8_v3"},
			file:         "azure-schema.json",
		},
		{
			name:         "AzureLite schema is correct",
			generator:    AzureLiteSchema,
			machineTypes: []string{"Standard_D4_v3"},
			file:         "azure-lite-schema.json",
		},
		{
			name:         "AzureHA schema is correct",
			generator:    AzureHASchema,
			machineTypes: []string{"Standard_D4_v3"},
			file:         "azure-ha-schema.json",
		},
		{
			name:         "GCP schema is correct",
			generator:    GCPSchema,
			machineTypes: []string{"n1-standard-2", "n1-standard-4", "n1-standard-8", "n1-standard-16", "n1-standard-32", "n1-standard-64"},
			file:         "gcp-schema.json",
		},
		{
			name:         "OpenStack schema is correct",
			generator:    OpenStackSchema,
			machineTypes: []string{"m1.large"},
			file:         "openstack-schema.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.generator(tt.machineTypes)
			validateSchema(t, got, tt.file)
		})
	}
}

func TestTrialSchemaGenerator(t *testing.T) {
	validateSchema(t, TrialSchema(), "azure-trial-schema.json")
}

func TestFreemiumAzureSchemaGenerator(t *testing.T) {
	validateSchema(t, FreemiumSchema(internal.Azure), "free-azure-schema.json")
}

func TestFreemiumAWSSchemaGenerator(t *testing.T) {
	validateSchema(t, FreemiumSchema(internal.AWS), "free-aws-schema.json")
}

func validateSchema(t *testing.T, got []byte, file string) {
	var prettyWant bytes.Buffer

	want := readJsonFile(t, file)
	err := json.Indent(&prettyWant, []byte(want), "", "  ")
	if err != nil {
		t.Error(err)
		t.Fail()
	}

	var prettyGot bytes.Buffer
	err = json.Indent(&prettyGot, got, "", "  ")
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	if !reflect.DeepEqual(prettyGot.String(), prettyWant.String()) {
		t.Errorf("Schema() = \n######### GOT ###########%v\n######### ENDGOT ########, want \n##### WANT #####%v\n##### ENDWANT #####", prettyGot.String(), prettyWant.String())
	}
}

func readJsonFile(t *testing.T, file string) string {
	t.Helper()

	filename := path.Join("testdata", file)
	yamlFile, err := ioutil.ReadFile(filename)
	require.NoError(t, err)

	return string(yamlFile)
}
