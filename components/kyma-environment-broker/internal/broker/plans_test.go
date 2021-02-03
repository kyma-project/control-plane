package broker

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestSchemaGenerator(t *testing.T) {
	tests := []struct {
		name         string
		generator    func([]string) []byte
		machineTypes []string
		want         string
	}{
		{
			name:         "Azure schema is correct",
			generator:    AzureSchema,
			machineTypes: []string{"Standard_D8_v3"},
			want: `{
			"$schema": "http://json-schema.org/draft-04/schema#",
			"type": "object",
			"properties": {
			"name": {
			"type": "string",
			"minLength": 1
		},
			"machineType": {
			"type": "string",
			"enum": ["Standard_D8_v3"]
		},
			"region": {
			"type": "string",
			"enum": [ "eastus", "centralus", "westus2", "uksouth", "northeurope", "westeurope", "japaneast", "southeastasia" ]
		},
			"autoScalerMin": {
			"type": "integer",
			"minimum": 2
		},
			"autoScalerMax": {
			"type": "integer",
			"minimum": 2,
            "maximum": 40,
            "default": 10
		}},
			"required": [
			"name"
		],
           "_show_form_view": true
		}`},
		{
			name:         "AzureLite schema is correct",
			generator:    AzureSchema,
			machineTypes: []string{"Standard_D4_v3"},
			want: `{
			"$schema": "http://json-schema.org/draft-04/schema#",
			"type": "object",
			"properties": {
			"name": {
			"type": "string",
			"minLength": 1
		},
			"machineType": {
			"type": "string",
			"enum": ["Standard_D4_v3"]
		},
			"region": {
			"type": "string",
			"enum": [ "eastus", "centralus", "westus2", "uksouth", "northeurope", "westeurope", "japaneast", "southeastasia" ]
		},
			"autoScalerMin": {
			"type": "integer",
			"minimum": 2
		},
			"autoScalerMax": {
			"type": "integer",
			"minimum": 2,
            "maximum": 40,
            "default": 10
		}},
			"required": [
			"name"
		],
           "_show_form_view": true
		}`},
		{
			name:         "GCP schema is correct",
			generator:    GCPSchema,
			machineTypes: []string{"n1-standard-2", "n1-standard-4", "n1-standard-8", "n1-standard-16", "n1-standard-32", "n1-standard-64"},
			want: `{
			"$schema": "http://json-schema.org/draft-04/schema#",
			"type": "object",
			"properties": {
			"name": {
			"type": "string",
			"minLength": 1
		},
			"machineType": {
			"type": "string",
			"enum": ["n1-standard-2", "n1-standard-4", "n1-standard-8", "n1-standard-16", "n1-standard-32", "n1-standard-64"]
		},
			"region": {
			"type": "string",
			"enum": ["asia-south1", "asia-southeast1",
					"asia-east2", "asia-east1",
					"asia-northeast1", "asia-northeast2", "asia-northeast-3",
					"australia-southeast1",
					"europe-west2", "europe-west4", "europe-west5", "europe-west6", "europe-west3",
					"europe-north1",
					"us-west1", "us-west2", "us-west3",
					"us-central1",
					"us-east4",
					"northamerica-northeast1", "southamerica-east1"]
		},
			"autoScalerMin": {
			"type": "integer",
			"minimum": 2
		},
			"autoScalerMax": {
			"type": "integer",
			"minimum": 2,
            "maximum": 40,
            "default": 10
		}},
			"required": [
			"name"
		],
           "_show_form_view": true
		}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.generator(tt.machineTypes)
			validateSchema(t, got, tt.want)
		})
	}
}

func TestTrialSchemaGenerator(t *testing.T) {
	want := `{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
	  "minLength": 1
    }
  },
  "required": [
    "name"
  ],
  "_show_form_view": true
}`

	got := TrialSchema()
	validateSchema(t, got, want)

}

func validateSchema(t *testing.T, got []byte, want string) {
	var prettyWant bytes.Buffer
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
	if !reflect.DeepEqual(string(prettyGot.String()), prettyWant.String()) {
		t.Errorf("Schema() = \n######### GOT ###########%v\n######### ENDGOT ########, want \n##### WANT #####%v\n##### ENDWANT #####", prettyGot.String(), prettyWant.String())
	}
}
