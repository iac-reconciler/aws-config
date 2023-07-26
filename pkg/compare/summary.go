package compare

import "sort"

type ResourceTypeCount struct {
	ResourceType string
	Count        int
}

type SourceSummary struct {
	Name            string
	Total           int
	Unmapped        []ResourceTypeCount
	UnmappedCount   int
	Only            []ResourceTypeCount
	OnlyCount       int
	OnlyMapped      []ResourceTypeCount
	OnlyMappedCount int
}

// Summary struct holding summary information about the various resources.
// This is expected to evolve over time.
type Summary struct {
	Sources       []SourceSummary
	BothResources int
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
	processSummaries(&terraform, terraformUnmapped, terraformOnly)
	results.Sources = append(results.Sources, terraform)

	processSummaries(&config, configUnmapped, configOnly)
	results.Sources = append(results.Sources, config)

	return results, nil
}

func processSummaries(sourceSummary *SourceSummary, unmapped map[string]int, only map[string]int) {
	for k, v := range unmapped {
		sourceSummary.Unmapped = append(sourceSummary.Unmapped, ResourceTypeCount{
			ResourceType: k,
			Count:        v,
		})
		sourceSummary.UnmappedCount += v
	}
	sort.Slice(sourceSummary.Unmapped, func(i, j int) bool {
		return sourceSummary.Unmapped[i].ResourceType < sourceSummary.Unmapped[j].ResourceType
	})
	for k, v := range only {
		sourceSummary.Only = append(sourceSummary.Only, ResourceTypeCount{
			ResourceType: k,
			Count:        v,
		})
		sourceSummary.OnlyCount += v
	}
	sort.Slice(sourceSummary.Only, func(i, j int) bool {
		return sourceSummary.Only[i].ResourceType < sourceSummary.Only[j].ResourceType
	})

}
