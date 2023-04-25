package cfngen

import (
	"testing"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/common-fate/pdk/pkg/iamp"
	"github.com/common-fate/pdk/pkg/pythonconfig"
)

func TestGenerateAccessRole(t *testing.T) {
	type args struct {
		pconfig  pythonconfig.Config
		roleName string
		policy   iamp.Policy
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "ok",
			args: args{
				pconfig: pythonconfig.Config{
					Publisher: "common-fate",
					Name:      "test-provider",
				},
				roleName: "cloudwatch-read",
				policy: iamp.NewPolicy(iamp.Statement{
					Effect:   iamp.Allow,
					Action:   iamp.Value{"s3:ListBucket"},
					Resource: iamp.Value{"*"},
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateAccessRole(tt.args.pconfig, tt.args.roleName, tt.args.policy)
			if err != nil {
				t.Fatal(err)
			}

			cupaloy.SnapshotT(t, got)
		})
	}
}
