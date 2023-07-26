package compare

import "sort"

type ResourceTypeCount struct {
	ResourceType string `json:"resource_type"`
	Count        int    `json:"count"`
}

// Summary struct holding summary information about the various resources.
// This is expected to evolve over time.
type Summary struct {
	TerraformResources int `json:"terraform_resource_count"`
	TerraformUnmapped  []ResourceTypeCount
	ConfigResources    int `json:"config_resource_count"`
	ConfigUnmapped     []ResourceTypeCount
	BothResources      int `json:"both_resource_count"`
}

// Summarize summarize the information from the reconciliation.
func Summarize(items []*LocatedItem) (results *Summary, err error) {
	results = &Summary{}
	terraformUnmapped := make(map[string]int)
	configUnmapped := make(map[string]int)
	for _, item := range items {
		if item.terraform {
			results.TerraformResources++
			if !item.mappedType {
				if _, ok := terraformUnmapped[item.ResourceType]; !ok {
					terraformUnmapped[item.ResourceType] = 0
				}
				terraformUnmapped[item.ResourceType]++
			}
		}
		if item.config {
			results.ConfigResources++
			if !item.mappedType {
				if _, ok := configUnmapped[item.ResourceType]; !ok {
					configUnmapped[item.ResourceType] = 0
				}
				configUnmapped[item.ResourceType]++
			}
		}
		if item.config && item.terraform {
			results.BothResources++
		}
	}
	for k, v := range terraformUnmapped {
		results.TerraformUnmapped = append(results.TerraformUnmapped, ResourceTypeCount{
			ResourceType: k,
			Count:        v,
		})
	}
	for k, v := range configUnmapped {
		results.ConfigUnmapped = append(results.ConfigUnmapped, ResourceTypeCount{
			ResourceType: k,
			Count:        v,
		})
	}
	// and now to sort each one
	sort.Slice(results.TerraformUnmapped, func(i, j int) bool {
		return results.TerraformUnmapped[i].ResourceType < results.TerraformUnmapped[j].ResourceType
	})
	sort.Slice(results.ConfigUnmapped, func(i, j int) bool {
		return results.ConfigUnmapped[i].ResourceType < results.ConfigUnmapped[j].ResourceType
	})
	return results, nil
}
