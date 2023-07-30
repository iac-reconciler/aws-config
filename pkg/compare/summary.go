package compare

import "sort"

const (
	sourceBeanstalk      = "beanstalk"
	sourceEKS            = "eks"
	sourceTerraform      = "terraform"
	sourceConfig         = "config"
	sourceCloudFormation = "cloudformation"
)

var (
	SourceKeys = []string{sourceBeanstalk, sourceEKS, sourceTerraform, sourceConfig, sourceCloudFormation}
)

func init() {
	sort.Strings(SourceKeys)
}

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
	SingleOnly   int
	Both         int
}

// Summary struct holding summary information about the various resources.
// This is expected to evolve over time.
type Summary struct {
	// ByType map by each type, with the values showing how many there
	// are in each source, total, and both
	ByType          []TypeSummary
	Sources         []SourceSummary
	BothResources   int
	SingleResources int
}

// Summarize summarize the information from the reconciliation.
func Summarize(items []*LocatedItem) (results *Summary, err error) {
	results = &Summary{}
	var (
		terraform     = SourceSummary{Name: sourceTerraform}
		config        = SourceSummary{Name: sourceConfig}
		cfn           = SourceSummary{Name: sourceCloudFormation}
		beanstalk     = SourceSummary{Name: sourceBeanstalk}
		eks           = SourceSummary{Name: sourceEKS}
		only          map[string]*ResourceTypeCount
		rtc           *ResourceTypeCount
		configOnly    = make(map[string]*ResourceTypeCount)
		terraformOnly = make(map[string]*ResourceTypeCount)
		cfnOnly       = make(map[string]*ResourceTypeCount)
		beanstalkOnly = make(map[string]*ResourceTypeCount)
		eksOnly       = make(map[string]*ResourceTypeCount)
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
			if _, ok := ts.Source[sourceTerraform]; !ok {
				ts.Source[sourceTerraform] = 0
			}
			ts.Source[sourceTerraform]++
			only = terraformOnly
		}
		if item.config {
			config.Total++
			if _, ok := ts.Source[sourceConfig]; !ok {
				ts.Source[sourceConfig] = 0
			}
			ts.Source[sourceConfig]++
			only = configOnly
		}
		if item.cfn {
			cfn.Total++
			if _, ok := ts.Source[sourceCloudFormation]; !ok {
				ts.Source[sourceCloudFormation] = 0
			}
			ts.Source[sourceCloudFormation]++
			only = cfnOnly
		}
		if item.beanstalk {
			beanstalk.Total++
			if _, ok := ts.Source[sourceBeanstalk]; !ok {
				ts.Source[sourceBeanstalk] = 0
			}
			ts.Source[sourceBeanstalk]++
			only = beanstalkOnly
		}
		if item.eks {
			eks.Total++
			if _, ok := ts.Source[sourceEKS]; !ok {
				ts.Source[sourceEKS] = 0
			}
			ts.Source[sourceEKS]++
			only = eksOnly
		}

		if _, ok := only[item.ResourceType]; !ok {
			only[item.ResourceType] = &ResourceTypeCount{
				ResourceType: item.ResourceType,
			}
		}
		rtc = only[item.ResourceType]
		if item.config && (item.terraform || item.cfn || item.beanstalk || item.eks) {
			ts.Both++
			results.BothResources++
		} else {
			ts.SingleOnly++
			results.SingleResources++
		}
		if item.mappedType {

		} else {
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

	processSummaries(&eks, eksOnly)
	results.Sources = append(results.Sources, eks)

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
