package compare

import (
	"github.com/iac-reconciler/tf-aws-config/pkg/load"
)

// Summary struct holding summary information about the various resources.
// This is expected to evolve over time.
type Summary struct {
	TerraformResources int `json:"terraform_resource_count"`
	TerraformFiles     int `json:"terraform_file_count"`
	ConfigResources    int `json:"config_resource_count"`
}

// Summarize summarize the information from the reconciliation.
func Summarize(snapshot load.Snapshot, tfstates []load.TerraformState) (results *Summary, err error) {
	var tfResources int
	for _, tfstate := range tfstates {
		tfResources += len(tfstate.Resources)
	}
	return &Summary{
		ConfigResources:    len(snapshot.ConfigurationItems),
		TerraformResources: tfResources,
		TerraformFiles:     len(tfstates),
	}, nil
}
