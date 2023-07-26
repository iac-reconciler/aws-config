package compare

import "sort"

type ResourceTypeCount struct {
	ResourceType string `json:"resource_type"`
	Count        int    `json:"count"`
}

// Summary struct holding summary information about the various resources.
// This is expected to evolve over time.
type Summary struct {
	TerraformResources         int `json:"terraform_resource_count"`
	TerraformUnmapped          []ResourceTypeCount
	TerraformUnmappedResources int
	TerraformOnly              []ResourceTypeCount
	TerraformOnlyResources     int
	ConfigResources            int `json:"config_resource_count"`
	ConfigUnmapped             []ResourceTypeCount
	ConfigUnmappedResources    int
	ConfigOnly                 []ResourceTypeCount
	ConfigOnlyResources        int
	BothResources              int `json:"both_resource_count"`
}

// Summarize summarize the information from the reconciliation.
func Summarize(items []*LocatedItem) (results *Summary, err error) {
	results = &Summary{}
	var (
		unmapped          map[string]int
		terraformUnmapped = make(map[string]int)
		configUnmapped    = make(map[string]int)
		only              map[string]int
		terraformOnly     = make(map[string]int)
		configOnly        = make(map[string]int)
	)
	for _, item := range items {
		if item.terraform {
			results.TerraformResources++
			unmapped = terraformUnmapped
			only = terraformOnly
		}
		if item.config {
			results.ConfigResources++
			unmapped = configUnmapped
			only = configOnly
		}
		if item.config && item.terraform {
			results.BothResources++
		} else {
			if _, ok := only[item.ResourceType]; !ok {
				only[item.ResourceType] = 0
			}
			only[item.ResourceType]++
		}
		if !item.mappedType {
			if _, ok := unmapped[item.ResourceType]; !ok {
				unmapped[item.ResourceType] = 0
			}
			unmapped[item.ResourceType]++
		}
	}
	// get summary by resource type for unmapped in terraform and unmapped in config
	for k, v := range terraformUnmapped {
		results.TerraformUnmapped = append(results.TerraformUnmapped, ResourceTypeCount{
			ResourceType: k,
			Count:        v,
		})
		results.TerraformUnmappedResources += v
	}
	for k, v := range configUnmapped {
		results.ConfigUnmapped = append(results.ConfigUnmapped, ResourceTypeCount{
			ResourceType: k,
			Count:        v,
		})
		results.ConfigUnmappedResources += v
	}
	// and now to sort each one
	sort.Slice(results.TerraformUnmapped, func(i, j int) bool {
		return results.TerraformUnmapped[i].ResourceType < results.TerraformUnmapped[j].ResourceType
	})
	sort.Slice(results.ConfigUnmapped, func(i, j int) bool {
		return results.ConfigUnmapped[i].ResourceType < results.ConfigUnmapped[j].ResourceType
	})

	// get summary by resource type for only in terraform and only in config
	for k, v := range terraformOnly {
		results.TerraformOnly = append(results.TerraformOnly, ResourceTypeCount{
			ResourceType: k,
			Count:        v,
		})
		results.TerraformOnlyResources += v
	}
	for k, v := range configOnly {
		results.ConfigOnly = append(results.ConfigOnly, ResourceTypeCount{
			ResourceType: k,
			Count:        v,
		})
		results.ConfigOnlyResources += v
	}
	// and now to sort each one
	sort.Slice(results.TerraformOnly, func(i, j int) bool {
		return results.TerraformOnly[i].ResourceType < results.TerraformOnly[j].ResourceType
	})
	sort.Slice(results.ConfigOnly, func(i, j int) bool {
		return results.ConfigOnly[i].ResourceType < results.ConfigOnly[j].ResourceType
	})
	return results, nil
}
