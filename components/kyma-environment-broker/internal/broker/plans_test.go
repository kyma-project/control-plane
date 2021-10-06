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
		generator    func([]string) RootSchema
		machineTypes []string
		file         string
		updateFile   string
	}{
		{
			name:         "AWS schema is correct",
			generator:    AWSSchema,
			machineTypes: []string{"m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge"},
			file:         "aws-schema.json",
			updateFile:   "update-aws-schema.json",
		},
		{
			name:         "AWS HA schema is correct",
			generator:    AWSHASchema,
			machineTypes: []string{"m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge"},
			file:         "aws-ha-schema.json",
			updateFile:   "update-aws-ha-schema.json",
		},
		{
			name:         "Azure schema is correct",
			generator:    AzureSchema,
			machineTypes: []string{"Standard_D8_v3"},
			file:         "azure-schema.json",
			updateFile:   "update-azure-schema.json",
		},
		{
			name:         "AzureLite schema is correct",
			generator:    AzureLiteSchema,
			machineTypes: []string{"Standard_D4_v3"},
			file:         "azure-lite-schema.json",
			updateFile:   "update-azure-lite-schema.json",
		},
		{
			name:         "AzureHA schema is correct",
			generator:    AzureHASchema,
			machineTypes: []string{"Standard_D8_v3"},
			file:         "azure-ha-schema.json",
			updateFile:   "update-azure-ha-schema.json",
		},
		{
			name:         "GCP schema is correct",
			generator:    GCPSchema,
			machineTypes: []string{"n1-standard-2", "n1-standard-4", "n1-standard-8", "n1-standard-16", "n1-standard-32", "n1-standard-64"},
			file:         "gcp-schema.json",
			updateFile:   "update-gcp-schema.json",
		},
		{
			name:         "OpenStack schema is correct",
			generator:    OpenStackSchema,
			machineTypes: []string{"m1.large"},
			file:         "openstack-schema.json",
			updateFile:   "update-openstack-schema.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.generator(tt.machineTypes)
			rawSchema := marshalSchema(got)
			rawUpdateSchema := schemaForUpdate(got)
			validateSchema(t, rawSchema, tt.file)
			validateSchema(t, rawUpdateSchema, tt.updateFile)
		})
	}
}

func TestTrialSchemaGenerator(t *testing.T) {
	validateSchema(t, marshalSchema(TrialSchema()), "azure-trial-schema.json")
	validateSchema(t, schemaForUpdate(TrialSchema()), "update-azure-trial-schema.json")
}

func TestFreemiumAzureSchemaGenerator(t *testing.T) {
	validateSchema(t, marshalSchema(FreemiumSchema(internal.Azure)), "free-azure-schema.json")
	validateSchema(t, schemaForUpdate(FreemiumSchema(internal.Azure)), "update-free-azure-schema.json")
}

func TestFreemiumAWSSchemaGenerator(t *testing.T) {
	validateSchema(t, marshalSchema(FreemiumSchema(internal.AWS)), "free-aws-schema.json")
	validateSchema(t, schemaForUpdate(FreemiumSchema(internal.AWS)), "update-free-aws-schema.json")
}

func TestSchemaWithOIDC(t *testing.T) {
	tests := []struct {
		name         string
		generator    func([]string) RootSchema
		machineTypes []string
		file         string
		updateFile   string
	}{
		{
			name:         "AWS schema with OIDC is correct",
			generator:    AWSSchema,
			machineTypes: []string{"m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge"},
			file:         "aws-schema-oidc.json",
			updateFile:   "update-aws-schema-oidc.json",
		},
		{
			name:         "AWS HA schema with OIDC is correct",
			generator:    AWSHASchema,
			machineTypes: []string{"m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge"},
			file:         "aws-ha-schema-oidc.json",
			updateFile:   "update-aws-ha-schema-oidc.json",
		},
		{
			name:         "Azure schema with OIDC is correct",
			generator:    AzureSchema,
			machineTypes: []string{"Standard_D8_v3"},
			file:         "azure-schema-oidc.json",
			updateFile:   "update-azure-schema-oidc.json",
		},
		{
			name:         "AzureLite schema with OIDC is correct",
			generator:    AzureLiteSchema,
			machineTypes: []string{"Standard_D4_v3"},
			file:         "azure-lite-schema-oidc.json",
			updateFile:   "update-azure-lite-schema-oidc.json",
		},
		{
			name:         "AzureHA schema with OIDC is correct",
			generator:    AzureHASchema,
			machineTypes: []string{"Standard_D8_v3"},
			file:         "azure-ha-schema-oidc.json",
			updateFile:   "update-azure-ha-schema-oidc.json",
		},
		{
			name:         "GCP schema with OIDC is correct",
			generator:    GCPSchema,
			machineTypes: []string{"n1-standard-2", "n1-standard-4", "n1-standard-8", "n1-standard-16", "n1-standard-32", "n1-standard-64"},
			file:         "gcp-schema-oidc.json",
			updateFile:   "update-gcp-schema-oidc.json",
		},
		{
			name:         "OpenStack schema with OIDC is correct",
			generator:    OpenStackSchema,
			machineTypes: []string{"m1.large"},
			file:         "openstack-schema-oidc.json",
			updateFile:   "update-openstack-schema-oidc.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.generator(tt.machineTypes)
			includeOIDCSchema(&got)
			rawSchema := marshalSchema(got)
			rawUpdateSchema := schemaForUpdate(got)
			validateSchema(t, rawSchema, tt.file)
			validateSchema(t, rawUpdateSchema, tt.updateFile)
		})
	}
}

func TestTrialSchemaWithOIDC(t *testing.T) {
	schema := TrialSchema()
	includeOIDCSchema(&schema)
	validateSchema(t, marshalSchema(schema), "azure-trial-schema-oidc.json")
	validateSchema(t, schemaForUpdate(schema), "update-azure-trial-schema-oidc.json")
}

func TestFreemiumAzureSchemaWithOIDC(t *testing.T) {
	schema := FreemiumSchema(internal.Azure)
	includeOIDCSchema(&schema)
	validateSchema(t, marshalSchema(schema), "free-azure-schema-oidc.json")
	validateSchema(t, schemaForUpdate(schema), "update-free-azure-schema-oidc.json")
}

func TestFreemiumAWSSchemaWithOIDC(t *testing.T) {
	schema := FreemiumSchema(internal.AWS)
	includeOIDCSchema(&schema)
	validateSchema(t, marshalSchema(schema), "free-aws-schema-oidc.json")
	validateSchema(t, schemaForUpdate(schema), "update-free-aws-schema-oidc.json")
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
