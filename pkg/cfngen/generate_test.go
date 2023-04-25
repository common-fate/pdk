package cfngen

import (
	"encoding/json"
	"testing"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/bradleyjkemp/cupaloy"
	"github.com/common-fate/pdk/pkg/pythonconfig"
	"github.com/common-fate/provider-registry-sdk-go/pkg/providerregistrysdk"
)

func TestConvertToPascalCase(t *testing.T) {
	test := struct {
		want  string
		input string
	}{
		want:  "CamelCase",
		input: "camel_case",
	}

	got := ConvertToPascalCase(test.input)

	if got != test.want {
		t.Errorf("want %s got %s", test.want, got)
	}
}

func TestGenerate(t *testing.T) {
	testcases := []struct {
		name         string
		give         providerregistrysdk.Schema
		giveProvider pythonconfig.Config
	}{
		{
			name: "ok",
			giveProvider: pythonconfig.Config{
				Name:      "test",
				Publisher: "example-org",
			},
			give: providerregistrysdk.Schema{
				Config: &map[string]providerregistrysdk.Config{
					"api_url": {
						Type:        "string",
						Description: cloudformation.String("some usage"),
					},
					"api_key": {
						Type:        "string",
						Description: cloudformation.String("API key"),
						Secret:      cloudformation.Bool(true),
					},
				},
			},
		},
		{
			name: "no secrets",
			giveProvider: pythonconfig.Config{
				Name:      "test",
				Publisher: "example-org",
			},
			give: providerregistrysdk.Schema{
				Config: &map[string]providerregistrysdk.Config{
					"config_value": {
						Type: "string",
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Generate(tc.giveProvider, tc.give)
			if err != nil {
				t.Fatal(err)
			}
			var tmp map[string]any

			err = json.Unmarshal(got, &tmp)
			if err != nil {
				t.Fatal(err)
			}

			formatted, err := json.MarshalIndent(tmp, "", "  ")
			if err != nil {
				t.Fatal(err)
			}

			cupaloy.SnapshotT(t, formatted)
		})
	}
}
