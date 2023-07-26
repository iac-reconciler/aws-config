package compare

import "sort"

type ResourceTypeCount struct {
	ResourceType string `json:"resource_type"`
	Count        int    `json:"count"`
}

type SourceSummary struct {
	Name          string              `json:"source"`
	Total         int                 `json:"count"`
	Unmapped      []ResourceTypeCount `json:"unmapped"`
	UnmappedCount int                 `json:"unmapped_count"`
	Only          []ResourceTypeCount `json:"only"`
	OnlyCount     int                 `json:"only_count"`
}

// Summary struct holding summary information about the various resources.
// This is expected to evolve over time.
type Summary struct {
	Sources       []SourceSummary `json:"sources"`
	BothResources int             `json:"both_resource_count"`
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
		terraform         = SourceSummary{Name: "terraform"}
		config            = SourceSummary{Name: "config"}
	)
	for _, item := range items {
		if item.terraform {
			terraform.Total++
			unmapped = terraformUnmapped
			only = terraformOnly
		}
		if item.config {
			config.Total++
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
		terraform.Unmapped = append(terraform.Unmapped, ResourceTypeCount{
			ResourceType: k,
			Count:        v,
		})
		terraform.UnmappedCount += v
	}
	for k, v := range configUnmapped {
		config.Unmapped = append(config.Unmapped, ResourceTypeCount{
			ResourceType: k,
			Count:        v,
		})
		config.UnmappedCount += v
	}
	// and now to sort each one
	sort.Slice(terraform.Unmapped, func(i, j int) bool {
		return terraform.Unmapped[i].ResourceType < terraform.Unmapped[j].ResourceType
	})
	sort.Slice(config.Unmapped, func(i, j int) bool {
		return config.Unmapped[i].ResourceType < config.Unmapped[j].ResourceType
	})

	// get summary by resource type for only in terraform and only in config
	for k, v := range terraformOnly {
		terraform.Only = append(terraform.Only, ResourceTypeCount{
			ResourceType: k,
			Count:        v,
		})
		terraform.OnlyCount += v
	}
	for k, v := range configOnly {
		config.Only = append(config.Only, ResourceTypeCount{
			ResourceType: k,
			Count:        v,
		})
		config.OnlyCount += v
	}
	// and now to sort each one
	sort.Slice(terraform.Only, func(i, j int) bool {
		return terraform.Only[i].ResourceType < terraform.Only[j].ResourceType
	})
	sort.Slice(config.Only, func(i, j int) bool {
		return config.Only[i].ResourceType < config.Only[j].ResourceType
	})

	// and join them
	results.Sources = append(results.Sources, terraform)
	results.Sources = append(results.Sources, config)
	return results, nil
}
