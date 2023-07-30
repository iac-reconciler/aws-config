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

type TypeSummary struct {
	ResourceType string
	Count        int
	Source       map[string]int
}

// Summary struct holding summary information about the various resources.
// This is expected to evolve over time.
type Summary struct {
	// ByType map by each type, with the values showing how many there
	// are in each source, total, and both
	ByType        []TypeSummary
	Sources       []SourceSummary
	BothResources int
}

// Summarize summarize the information from the reconciliation.
func Summarize(items []*LocatedItem) (results *Summary, err error) {
	results = &Summary{}
	var (
		terraform     = SourceSummary{Name: "terraform"}
		config        = SourceSummary{Name: "config"}
		cfn           = SourceSummary{Name: "cloudformation"}
		beanstalk     = SourceSummary{Name: "beanstalk"}
		only          map[string]*ResourceTypeCount
		rtc           *ResourceTypeCount
		configOnly    = make(map[string]*ResourceTypeCount)
		terraformOnly = make(map[string]*ResourceTypeCount)
		cfnOnly       = make(map[string]*ResourceTypeCount)
		beanstalkOnly = make(map[string]*ResourceTypeCount)
		byType        = make(map[string]*TypeSummary)
	)
	for _, item := range items {
		if item.ConfigurationItem == nil {
			continue
		}
		if _, ok := byType[item.ResourceType]; !ok {
			byType[item.ResourceType] = &TypeSummary{
				ResourceType: item.ResourceType,
				Source:       make(map[string]int),
			}
		}
		ts := byType[item.ResourceType]
		ts.Count++
		if item.terraform {
			terraform.Total++
			if _, ok := ts.Source["terraform"]; !ok {
				ts.Source["terraform"] = 0
			}
			ts.Source["terraform"]++
			only = terraformOnly
		}
		if item.config {
			config.Total++
			if _, ok := ts.Source["config"]; !ok {
				ts.Source["config"] = 0
			}
			ts.Source["config"]++
			only = configOnly
		}
		if item.cfn {
			cfn.Total++
			if _, ok := ts.Source["cloudformation"]; !ok {
				ts.Source["cloudformation"] = 0
			}
			ts.Source["cloudformation"]++
			only = cfnOnly
		}
		if item.beanstalk {
			beanstalk.Total++
			if _, ok := ts.Source["beanstalk"]; !ok {
				ts.Source["beanstalk"] = 0
			}
			ts.Source["beanstalk"]++
			only = beanstalkOnly
		}
		if _, ok := only[item.ResourceType]; !ok {
			only[item.ResourceType] = &ResourceTypeCount{
				ResourceType: item.ResourceType,
			}
		}
		rtc = only[item.ResourceType]
		switch {
		case item.config && (item.terraform || item.cfn || item.beanstalk):
			if _, ok := ts.Source["both"]; !ok {
				ts.Source["both"] = 0
			}
			ts.Source["both"]++
			results.BothResources++
		case item.mappedType:
			rtc.Mapped++
		default:
			rtc.Unmapped++
		}
	}
	for _, v := range byType {
		results.ByType = append(results.ByType, *v)
	}
	sort.Slice(results.ByType, func(i, j int) bool {
		return results.ByType[i].ResourceType < results.ByType[j].ResourceType
	})
	// get summary by resource type for unmapped in terraform and unmapped in config
	processSummaries(&terraform, terraformOnly)
	results.Sources = append(results.Sources, terraform)

	processSummaries(&config, configOnly)
	results.Sources = append(results.Sources, config)

	processSummaries(&cfn, cfnOnly)
	results.Sources = append(results.Sources, cfn)

	processSummaries(&beanstalk, beanstalkOnly)
	results.Sources = append(results.Sources, beanstalk)

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
