package compare

import (
	"github.com/iac-reconciler/tf-aws-config/pkg/load"
)

// Summary struct holding summary information about the various resources.
// This is expected to evolve over time.
type Summary struct {
	TerraformResources int `json:"terraform_resource_count"`
	ConfigResources    int `json:"config_resource_count"`
}

// Reconcile reconcile the snapshot and tfstates.
// Not yet implemented, so returns an empty struct
func Summarize(snapshot load.Snapshot, tfstates []load.TerraformState) (results *Summary, err error) {
	var tfResources int
	for _, tfstate := range tfstates {
		tfResources += len(tfstate.Resources)
	}
	return &Summary{
		ConfigResources:    len(snapshot.ConfigurationItems),
		TerraformResources: tfResources,
	}, nil
}
