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
		Use:   "detail",
		Short: "show details for specific resources in the sources",
		Long: `Show detail for specific resources in the sources. By default, shows all resources.
		Can be restricted to just one or a few resource types.`,
		Example: `
		aws-config detail --aws-config <aws-config-snapshot.json> --terraform <terraform.tfstate>
		aws-config detail --aws-config <aws-config-snapshot.json> --terraform <terraform.tfstate> AWS::EC2::Volume
		aws-config detail --aws-config <aws-config-snapshot.json> --terraform <terraform.tfstate> AWS::EC2::Volume AWS::EC2::RouteTable
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := make(map[string]bool)
			for _, arg := range args {
				resource[arg] = true
			}
			hasRestrictions := len(resource) > 0
			var results []*compare.LocatedItem
			for _, item := range items {
				if item.Ephemeral() {
					continue
				}
				if !hasRestrictions || resource[item.ResourceType] {
					results = append(results, item)
				}
			}
			// print the detail
			sort.Slice(results, func(i, j int) bool {
				var retVal bool
				iType, jType := results[i].ResourceType, results[j].ResourceType
				iName, jName := results[i].ResourceName, results[j].ResourceName
				iID, jID := results[i].ResourceID, results[j].ResourceID
				if strings.HasPrefix(sortBy, "count-") {
					key := strings.TrimPrefix(sortBy, "count-")
					iValue := results[i].Source(key)
					jValue := results[j].Source(key)
					switch {
					case iValue == jValue:
						// if the values of the fields are the same, sort
						// by name or by ID
						if iName == jName {
							retVal = iID < jID
						} else {
							retVal = iName < jName
						}
					case iValue:
						retVal = true
					default:
						retVal = false
					}
				} else {
					retVal = iType < jType
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
