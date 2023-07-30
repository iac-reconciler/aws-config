package cli

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/iac-reconciler/aws-config/pkg/compare"
	"github.com/iac-reconciler/aws-config/pkg/load"
	"github.com/spf13/cobra"
)

func generate() *cobra.Command {
	var (
		tfRecursive               bool
		summarize, byResourceType bool
		descending                bool
		sortBy, detail            string
		top                       int
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
		Use:     "generate",
		Short:   "Generate a unified summary of AWS Config snapshot and terraform files",
		Long:    `Generate a unified summary of AWS Config snapshot and terraform files.`,
		Example: `  aws-config generate <aws-config-snapshot.json> <terraform.tfstate>`,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			snapshotFile := args[0]
			var (
				tfstate []string
				err     error
				fsys    fs.FS
			)
			if tfRecursive {
				fsys = os.DirFS(args[1])
				if err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
					if filepath.Ext(path) == ".tfstate" {
						tfstate = append(tfstate, path)
					}
					return nil
				}); err != nil {
					return err
				}
			} else {
				fsys = os.DirFS(path.Dir(args[1]))
				tfstate = []string{path.Base(args[1])}
			}
			// read the config file
			f, err := os.Open(snapshotFile)
			if err != nil {
				return fmt.Errorf("unable to open snapshot file %s: %w", snapshotFile, err)
			}
			defer f.Close()
			var snapshot load.Snapshot
			if err := json.NewDecoder(f).Decode(&snapshot); err != nil {
				return fmt.Errorf("unable to decode snapshot file %s: %w", snapshotFile, err)
			}
			// read the tfstate files
			var tfstates = make(map[string]load.TerraformState)
			for _, tfstateFile := range tfstate {
				f, err := fsys.Open(tfstateFile)
				if err != nil {
					return fmt.Errorf("unable to open tfstate file %s: %w", tfstateFile, err)
				}
				defer f.Close()
				var state load.TerraformState
				if err := json.NewDecoder(f).Decode(&state); err != nil {
					return fmt.Errorf("unable to decode terraform state file %s: %w", tfstateFile, err)
				}
				tfstates[tfstateFile] = state
			}
			// all loaded, now run the reconcile
			items, err := compare.Reconcile(snapshot, tfstates)
			if err != nil {
				return fmt.Errorf("unable to reconcile: %w", err)
			}
			summary, err := compare.Summarize(items)
			if err != nil {
				return fmt.Errorf("unable to summarize: %w", err)
			}

			if detail != "" {
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
						iValue := results[i].Value(key)
						jValue := results[j].Value(key)
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
				fmt.Printf("ResourceType ResourceName ResourceID ARN beanstalk %s\n", strings.Join(compare.SourceKeys, " "))
				for _, item := range results {
					if item.ResourceType != detail {
						continue
					}
					fmt.Printf("%s %s %s %s %v %v %v %v %v\n",
						item.ResourceType,
						item.ResourceName,
						item.ResourceID,
						item.ARN,
						item.Beanstalk(),
						item.CloudFormation(),
						item.Config(),
						item.EKS(),
						item.Terraform(),
					)
				}
				return nil
			}

			if byResourceType {
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

				fmt.Printf("ResourceType Total (Config-Only) (Config+IaC) %s\n", strings.Join(compare.SourceKeys, " "))
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
			}

			fmt.Println()
			if summarize {
				fmt.Printf("Summary:\n")
				fmt.Printf("Both (Config+IaC): %d\n", summary.BothResources)

				for _, source := range summary.Sources {
					fmt.Printf("%s:\n", source.Name)
					fmt.Printf("\tAll Resources: %d\n", source.Total)
					fmt.Printf("\tOnly in %s: %d\n", source.Name, source.OnlyCount)
					fmt.Printf("\t\tMapped (matching type in alternate): %d\n", source.OnlyMappedCount)
					fmt.Printf("\t\tUnmapped (no matching type in alternate): %d\n", source.OnlyUnmappedCount)
				}
				fmt.Printf("Terraform Files: %d\n", len(tfstates))
			}
			// no error
			return nil
		},
	}

	cmd.Flags().BoolVar(&tfRecursive, "tf-recursive", false, "treat the path to terraform state as a directory and recursively search for .tfstate files")
	cmd.Flags().BoolVar(&byResourceType, "by-type", false, "list the count of locations of each resource type")
	cmd.Flags().BoolVar(&summarize, "summary", false, "provide summary results")
	cmd.Flags().BoolVar(&descending, "descending", false, "sort by descending instead of ascending; for by-type and detail")
	cmd.Flags().StringVar(&sortBy, "sort", sortByDefault, "sort order for results, options are: "+strings.Join(sortOptions, " ")+", as well as 'count-<field>', where <field> is any supported field, e.g. terraform or eks; for by-type and detail")
	cmd.Flags().IntVar(&top, "top", 0, "limit to the top x results, use 0 for all, negative for last; for by-type and detail")
	cmd.Flags().StringVar(&detail, "detail", "", "report resource detail for a single resource type")
	return cmd
}
