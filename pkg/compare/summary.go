package compare

// Summary struct holding summary information about the various resources.
// This is expected to evolve over time.
type Summary struct {
	TerraformResources int `json:"terraform_resource_count"`
	ConfigResources    int `json:"config_resource_count"`
	BothResources      int `json:"both_resource_count"`
}

// Summarize summarize the information from the reconciliation.
func Summarize(items []*LocatedItem) (results *Summary, err error) {
	results = &Summary{}
	for _, item := range items {
		if item.terraform {
			results.TerraformResources++
		}
		if item.config {
			results.ConfigResources++
		}
		if item.config && item.terraform {
			results.BothResources++
		}
	}
	return results, nil
}
