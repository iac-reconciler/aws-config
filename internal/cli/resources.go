package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/iac-reconciler/aws-config/pkg/compare"
	"github.com/spf13/cobra"
)

func resources() *cobra.Command {
	var (
		descending bool
		sortBy     string
		top        int
	)

	const (
		sortByResourceName    = "resource-name"
		sortByCountTotal      = "count-total"
		sortByCountBoth       = "count-both"
		sortByCountSingleOnly = "count-single"
		sortByDefault         = sortByResourceName
	)
	var sortOptions = []string{
		sortByResourceName,
		sortByCountTotal,
		sortByCountBoth,
		sortByCountSingleOnly,
	}

	cmd := &cobra.Command{
		Use:     "resources",
		Short:   "show count of individual resource types in AWS Config snapshot and terraform files",
		Long:    `Show count of individual resource types in AWS Config snapshot and terraform files.`,
		Example: `  aws-config resources --aws-config <aws-config-snapshot.json> --terraform <terraform.tfstate>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, err := compare.Summarize(items)
			if err != nil {
				return fmt.Errorf("unable to summarize: %w", err)
			}

			// sort the summary
			sort.Slice(summary.ByType, func(i, j int) bool {
				var retVal bool
				switch sortBy {
				case sortByCountTotal:
					retVal = summary.ByType[i].Count < summary.ByType[j].Count
				case sortByCountBoth:
					retVal = summary.ByType[i].Both < summary.ByType[j].Both
				case sortByCountSingleOnly:
					retVal = summary.ByType[i].SingleOnly < summary.ByType[j].SingleOnly
				case sortByResourceName:
					retVal = summary.ByType[i].ResourceType < summary.ByType[j].ResourceType
				default:
					if strings.HasPrefix(sortBy, "count-") {
						key := strings.TrimPrefix(sortBy, "count-")
						retVal = summary.ByType[i].Source[key] < summary.ByType[j].Source[key]
					} else {
						retVal = summary.ByType[i].ResourceType < summary.ByType[j].ResourceType
					}
				}
				if descending {
					retVal = !retVal
				}
				return retVal
			})
			var results []compare.TypeSummary
			// if limited
			switch {
			case top > 0:
				results = summary.ByType[:top]
			case top < 0:
				results = summary.ByType[len(summary.ByType)+top:]
			default:
				results = summary.ByType
			}

			fmt.Printf("ResourceType Total Single-Only Both %s\n", strings.Join(compare.SourceKeys, " "))
			for _, item := range results {
				fmt.Printf("%s: %d %d %d ",
					item.ResourceType,
					item.Count,
					item.SingleOnly,
					item.Both,
				)
				for _, source := range compare.SourceKeys {
					fmt.Printf("%d ", item.Source[source])
				}
				fmt.Println()
			}

			// no error
			return nil
		},
	}

	cmd.Flags().BoolVar(&descending, "descending", false, "sort by descending instead of ascending; for by-type and detail")
	cmd.Flags().StringVar(&sortBy, "sort", sortByDefault, "sort order for results, options are: "+strings.Join(sortOptions, " ")+", as well as 'count-<field>', where <field> is any supported field, e.g. terraform or eks; for by-type and detail")
	cmd.Flags().IntVar(&top, "top", 0, "limit to the top x results, use 0 for all, negative for last; for by-type and detail")
	return cmd
}
