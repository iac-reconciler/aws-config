package compare

import "sort"

// ResourceTypeCount used to keep track of resources that are only in one
// source or the other. Tracks separate counts for mapped and unmapped.
type ResourceTypeCount struct {
	ResourceType string
	Unmapped     int
	Mapped       int
}

type SourceSummary struct {
	Name              string
	Total             int
	Only              []ResourceTypeCount
	OnlyCount         int
	OnlyMappedCount   int
	OnlyUnmappedCount int
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
		terraform     = SourceSummary{Name: "terraform"}
		config        = SourceSummary{Name: "config"}
		only          map[string]*ResourceTypeCount
		rtc           *ResourceTypeCount
		configOnly    = make(map[string]*ResourceTypeCount)
		terraformOnly = make(map[string]*ResourceTypeCount)
	)
	for _, item := range items {
		if item.terraform {
			terraform.Total++
			only = terraformOnly
		}
		if item.config {
			config.Total++
			only = configOnly
		}
		if _, ok := only[item.ResourceType]; !ok {
			only[item.ResourceType] = &ResourceTypeCount{
				ResourceType: item.ResourceType,
			}
		}
		rtc = only[item.ResourceType]
		switch {
		case item.config && item.terraform:
			results.BothResources++
		case item.mappedType:
			rtc.Mapped++
		default:
			rtc.Unmapped++
		}
	}
	// get summary by resource type for unmapped in terraform and unmapped in config
	processSummaries(&terraform, terraformOnly)
	results.Sources = append(results.Sources, terraform)

	processSummaries(&config, configOnly)
	results.Sources = append(results.Sources, config)

	return results, nil
}

func processSummaries(sourceSummary *SourceSummary, only map[string]*ResourceTypeCount) {
	for _, v := range only {
		sourceSummary.Only = append(sourceSummary.Only, *v)
		sourceSummary.OnlyUnmappedCount += v.Unmapped
		sourceSummary.OnlyMappedCount += v.Mapped
		sourceSummary.OnlyCount += v.Unmapped + v.Mapped
	}
	sort.Slice(sourceSummary.Only, func(i, j int) bool {
		return sourceSummary.Only[i].ResourceType < sourceSummary.Only[j].ResourceType
	})
}
