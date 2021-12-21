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
		name           string
		generator      func([]string) RootSchema
		machineTypes   []string
		file           string
		updateFile     string
		fileOIDC       string
		updateFileOIDC string
	}{
		{
			name:           "AWS schema is correct",
			generator:      AWSSchema,
			machineTypes:   []string{"m6i.2xlarge", "m6i.4xlarge", "m6i.8xlarge", "m6i.12xlarge"},
			file:           "aws-schema.json",
			updateFile:     "update-aws-schema.json",
			fileOIDC:       "aws-schema-additional-params.json",
			updateFileOIDC: "update-aws-schema-additional-params.json",
		},
		{
			name:           "AWS HA schema is correct",
			generator:      AWSHASchema,
			machineTypes:   []string{"m6i.2xlarge", "m6i.4xlarge", "m6i.8xlarge", "m6i.12xlarge"},
			file:           "aws-ha-schema.json",
			updateFile:     "update-aws-ha-schema.json",
			fileOIDC:       "aws-ha-schema-additional-params.json",
			updateFileOIDC: "update-aws-ha-schema-additional-params.json",
		},
		{
			name:           "Azure schema is correct",
			generator:      AzureSchema,
			machineTypes:   []string{"Standard_D8_v3"},
			file:           "azure-schema.json",
			updateFile:     "update-azure-schema.json",
			fileOIDC:       "azure-schema-additional-params.json",
			updateFileOIDC: "update-azure-schema-additional-params.json",
		},
		{
			name:           "AzureLite schema is correct",
			generator:      AzureLiteSchema,
			machineTypes:   []string{"Standard_D4_v3"},
			file:           "azure-lite-schema.json",
			updateFile:     "update-azure-lite-schema.json",
			fileOIDC:       "azure-lite-schema-additional-params.json",
			updateFileOIDC: "update-azure-lite-schema-additional-params.json",
		},
		{
			name:           "AzureHA schema is correct",
			generator:      AzureHASchema,
			machineTypes:   []string{"Standard_D8_v3"},
			file:           "azure-ha-schema.json",
			updateFile:     "update-azure-ha-schema.json",
			fileOIDC:       "azure-ha-schema-additional-params.json",
			updateFileOIDC: "update-azure-ha-schema-additional-params.json",
		},
		{
			name:           "GCP schema is correct",
			generator:      GCPSchema,
			machineTypes:   []string{"n2-standard-8", "n2-standard-16", "n2-standard-32", "n2-standard-48"},
			file:           "gcp-schema.json",
			updateFile:     "update-gcp-schema.json",
			fileOIDC:       "gcp-schema-additional-params.json",
			updateFileOIDC: "update-gcp-schema-additional-params.json",
		},
		{
			name:           "OpenStack schema is correct",
			generator:      OpenStackSchema,
			machineTypes:   []string{"m1.large"},
			file:           "openstack-schema.json",
			updateFile:     "update-openstack-schema.json",
			fileOIDC:       "openstack-schema-additional-params.json",
			updateFileOIDC: "update-openstack-schema-additional-params.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.generator(tt.machineTypes)
			validateSchema(t, marshalSchema(got), tt.file)
			validateSchema(t, schemaForUpdate(got), tt.updateFile)
			includeOIDCSchema(&got)
			includeAdminsSchema(&got)
			validateSchema(t, marshalSchema(got), tt.fileOIDC)
			validateSchema(t, schemaForUpdate(got), tt.updateFileOIDC)
		})
	}
}

func TestTrialSchemaGenerator(t *testing.T) {
	schema := TrialSchema()
	validateSchema(t, marshalSchema(schema), "azure-trial-schema.json")
	validateSchema(t, schemaForUpdate(schema), "update-azure-trial-schema.json")
	includeOIDCSchema(&schema)
	includeAdminsSchema(&schema)
	validateSchema(t, marshalSchema(schema), "azure-trial-schema-additional-params.json")
	validateSchema(t, schemaForUpdate(schema), "update-azure-trial-schema-additional-params.json")
}

func TestFreemiumAzureSchemaGenerator(t *testing.T) {
	schema := FreemiumSchema(internal.Azure)
	validateSchema(t, marshalSchema(schema), "free-azure-schema.json")
	validateSchema(t, schemaForUpdate(schema), "update-free-azure-schema.json")
	includeOIDCSchema(&schema)
	includeAdminsSchema(&schema)
	validateSchema(t, marshalSchema(schema), "free-azure-schema-additional-params.json")
	validateSchema(t, schemaForUpdate(schema), "update-free-azure-schema-additional-params.json")
}

func TestFreemiumAWSSchemaGenerator(t *testing.T) {
	schema := FreemiumSchema(internal.AWS)
	validateSchema(t, marshalSchema(schema), "free-aws-schema.json")
	validateSchema(t, schemaForUpdate(schema), "update-free-aws-schema.json")
	includeOIDCSchema(&schema)
	includeAdminsSchema(&schema)
	validateSchema(t, marshalSchema(schema), "free-aws-schema-additional-params.json")
	validateSchema(t, schemaForUpdate(schema), "update-free-aws-schema-additional-params.json")
}

func validateSchema(t *testing.T, got []byte, file string) {
	var prettyWant bytes.Buffer
	want := readJsonFile(t, file)
	if len(want) > 0 {
		err := json.Indent(&prettyWant, []byte(want), "", "  ")
		if err != nil {
			t.Error(err)
			t.Fail()
		}
	}

	var prettyGot bytes.Buffer
	if len(got) > 0 {
		err := json.Indent(&prettyGot, got, "", "  ")
		if err != nil {
			t.Error(err)
			t.Fail()
		}
	}
	if !reflect.DeepEqual(prettyGot.String(), prettyWant.String()) {
		t.Errorf("%v Schema() = \n######### GOT ###########%v\n######### ENDGOT ########, want \n##### WANT #####%v\n##### ENDWANT #####", file, prettyGot.String(), prettyWant.String())
	}
}

func readJsonFile(t *testing.T, file string) string {
	t.Helper()

	filename := path.Join("testdata", file)
	yamlFile, err := ioutil.ReadFile(filename)
	require.NoError(t, err)

	return string(yamlFile)
}
