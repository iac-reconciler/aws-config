package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/iac-reconciler/aws-config/pkg/compare"
	"github.com/spf13/cobra"
)

func detail() *cobra.Command {
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
		sortByCountOwned      = "count-owned"
		sortByDefault         = sortByResourceName
	)
	var sortOptions = []string{
		sortByResourceName,
		sortByCountTotal,
		sortByCountBoth,
		sortByCountSingleOnly,
		sortByCountOwned,
	}

	cmd := &cobra.Command{
		Use:     "detail",
		Short:   "show detail for a specific resource in the sources",
		Long:    `Show detail for a specific resource in the sources.`,
		Example: `  aws-config detail --aws-config <aws-config-snapshot.json> --terraform <terraform.tfstate> AWS::EC2::Volume`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			detail := args[0]
			var results []*compare.LocatedItem
			for _, item := range items {
				if item.ResourceType != detail {
					continue
				}
				results = append(results, item)
			}
			// print the detail
			sort.Slice(results, func(i, j int) bool {
				var retVal bool
				if strings.HasPrefix(sortBy, "count-") {
					key := strings.TrimPrefix(sortBy, "count-")
					iValue := results[i].Source(key)
					jValue := results[j].Source(key)
					switch {
					case iValue == jValue:
						retVal = true
					case iValue:
						retVal = true
					default:
						retVal = false
					}
				} else {
					retVal = results[i].ResourceType < results[j].ResourceType
				}
				if descending {
					retVal = !retVal
				}
				return retVal
			})
			switch {
			case top > 0:
				results = results[:top]
			case top < 0:
				results = results[len(results)+top:]
			}
			fmt.Printf("ResourceType ResourceName ResourceID ARN owned %s\n", strings.Join(compare.SourceKeys, " "))
			for _, item := range results {
				if item.ResourceType != detail || item.Ephemeral() {
					continue
				}
				var line strings.Builder
				line.WriteString(fmt.Sprintf("%s %s %s %s %v",
					item.ResourceType,
					item.ResourceName,
					item.ResourceID,
					item.ARN,
					item.Owned(),
				))
				for _, key := range compare.SourceKeys {
					line.WriteString(fmt.Sprintf(" %v", item.Source(key)))
				}
				fmt.Println(line.String())
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
