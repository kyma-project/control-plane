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
		name                string
		generator           func(map[string]string, []string, bool, bool) *map[string]interface{}
		machineTypes        []string
		machineTypesDisplay map[string]string
		file                string
		updateFile          string
		fileOIDC            string
		updateFileOIDC      string
	}{
		{
			name:           "AWS schema is correct",
			generator:      AWSSchema,
			machineTypes:   []string{"m5.xlarge", "m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge", "m6i.xlarge", "m6i.2xlarge", "m6i.4xlarge", "m6i.8xlarge", "m6i.12xlarge"},
			file:           "aws-schema.json",
			updateFile:     "update-aws-schema.json",
			fileOIDC:       "aws-schema-additional-params.json",
			updateFileOIDC: "update-aws-schema-additional-params.json",
		},
		{
			name:           "Azure schema is correct",
			generator:      AzureSchema,
			machineTypes:   []string{"Standard_D4_v3", "Standard_D8_v3", "Standard_D16_v3", "Standard_D32_v3", "Standard_D48_v3", "Standard_D64_v3"},
			file:           "azure-schema.json",
			updateFile:     "update-azure-schema.json",
			fileOIDC:       "azure-schema-additional-params.json",
			updateFileOIDC: "update-azure-schema-additional-params.json",
		},
		{
			name:                "AzureLite schema is correct",
			generator:           AzureLiteSchema,
			machineTypes:        []string{"Standard_D4_v3"},
			machineTypesDisplay: map[string]string{"Standard_D4_v3": "Standard_D4_v3 (4vCPU, 16GB RAM)"},
			file:                "azure-lite-schema.json",
			updateFile:          "update-azure-lite-schema.json",
			fileOIDC:            "azure-lite-schema-additional-params.json",
			updateFileOIDC:      "update-azure-lite-schema-additional-params.json",
		},
		{
			name:           "GCP schema is correct",
			generator:      GCPSchema,
			machineTypes:   []string{"n2-standard-4", "n2-standard-8", "n2-standard-16", "n2-standard-32", "n2-standard-48"},
			file:           "gcp-schema.json",
			updateFile:     "update-gcp-schema.json",
			fileOIDC:       "gcp-schema-additional-params.json",
			updateFileOIDC: "update-gcp-schema-additional-params.json",
		},
		{
			name:           "OpenStack schema is correct",
			generator:      OpenStackSchema,
			machineTypes:   []string{"g_c4_m16", "g_c8_m32"},
			file:           "openstack-schema.json",
			updateFile:     "update-openstack-schema.json",
			fileOIDC:       "openstack-schema-additional-params.json",
			updateFileOIDC: "update-openstack-schema-additional-params.json",
		},
		{
			name: "Trial schema is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return TrialSchema(additionalParams, update)
			},
			machineTypes:   []string{},
			file:           "azure-trial-schema.json",
			updateFile:     "update-azure-trial-schema.json",
			fileOIDC:       "azure-trial-schema-additional-params.json",
			updateFileOIDC: "update-azure-trial-schema-additional-params.json",
		},
		{
			name: "Own cluster schema is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return OwnClusterSchema(additionalParams, update)
			},
			machineTypes:   []string{},
			file:           "own-cluster-schema.json",
			updateFile:     "update-own-cluster-schema.json",
			fileOIDC:       "own-cluster-schema-additional-params.json",
			updateFileOIDC: "update-own-cluster-schema-additional-params.json",
		},
		{
			name: "Freemium schema is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return FreemiumSchema(internal.Azure, additionalParams, update)
			},
			machineTypes:   []string{},
			file:           "free-azure-schema.json",
			updateFile:     "update-free-azure-schema.json",
			fileOIDC:       "free-azure-schema-additional-params.json",
			updateFileOIDC: "update-free-azure-schema-additional-params.json",
		},

		{
			name: " schema is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return FreemiumSchema(internal.AWS, additionalParams, update)
			},
			machineTypes:   []string{},
			file:           "free-aws-schema.json",
			updateFile:     "update-free-aws-schema.json",
			fileOIDC:       "free-aws-schema-additional-params.json",
			updateFileOIDC: "update-free-aws-schema-additional-params.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.generator(tt.machineTypesDisplay, tt.machineTypes, false, false)
			validateSchema(t, Marshal(got), tt.file)

			got = tt.generator(tt.machineTypesDisplay, tt.machineTypes, false, true)
			validateSchema(t, Marshal(got), tt.updateFile)

			got = tt.generator(tt.machineTypesDisplay, tt.machineTypes, true, false)
			validateSchema(t, Marshal(got), tt.fileOIDC)

			got = tt.generator(tt.machineTypesDisplay, tt.machineTypes, true, true)
			validateSchema(t, Marshal(got), tt.updateFileOIDC)
		})
	}
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
