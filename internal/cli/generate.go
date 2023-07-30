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

	"github.com/iac-reconciler/tf-aws-config/pkg/compare"
	"github.com/iac-reconciler/tf-aws-config/pkg/load"
	"github.com/spf13/cobra"
)

func generate() *cobra.Command {
	var (
		tfRecursive               bool
		summarize, byResourceType bool
		descending                bool
		sortBy                    string
		top                       int
	)

	const (
		sortByResourceName        = "resource-name"
		sortByCountTotal          = "count-total"
		sortByCountConfig         = "count-config"
		sortByCountTerraform      = "count-terraform"
		sortByCountCloudFormation = "count-cloudformation"
		sortByCountBeanstalk      = "count-beanstalk"
		sortByCountBoth           = "count-both"
		sortByCountSingleOnly     = "count-single"
		sortByDefault             = sortByResourceName
	)
	var sortOptions = []string{
		sortByResourceName,
		sortByCountTotal,
		sortByCountConfig,
		sortByCountTerraform,
		sortByCountCloudFormation,
		sortByCountBeanstalk,
	}

	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate a unified summary of AWS Config snapshot and terraform files",
		Long:    `Generate a unified summary of AWS Config snapshot and terraform files.`,
		Example: `  tf-aws-config generate <aws-config-snapshot.json> <terraform.tfstate>`,
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
			if byResourceType {
				fmt.Printf("ResourceType Total Config Terraform CFN Beanstalk EKS (Config+IaC)\n")
				// sort the summary
				sort.Slice(summary.ByType, func(i, j int) bool {
					var retVal bool
					switch sortBy {
					case sortByCountTotal:
						retVal = summary.ByType[i].Count < summary.ByType[j].Count
					case sortByCountConfig:
						retVal = summary.ByType[i].Source["config"] < summary.ByType[j].Source["config"]
					case sortByCountTerraform:
						retVal = summary.ByType[i].Source["terraform"] < summary.ByType[j].Source["terraform"]
					case sortByCountCloudFormation:
						retVal = summary.ByType[i].Source["cloudformation"] < summary.ByType[j].Source["cloudformation"]
					case sortByCountBeanstalk:
						retVal = summary.ByType[i].Source["beanstalk"] < summary.ByType[j].Source["beanstalk"]
					case sortByCountBoth:
						retVal = summary.ByType[i].Source["both"] < summary.ByType[j].Source["both"]
					case sortByCountSingleOnly:
						retVal = summary.ByType[i].Source["single-only"] < summary.ByType[j].Source["single-only"]
					case sortByResourceName:
						retVal = summary.ByType[i].ResourceType < summary.ByType[j].ResourceType
					default:
						retVal = summary.ByType[i].ResourceType < summary.ByType[j].ResourceType
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

				for _, item := range results {
					fmt.Printf("%s: %d %d %d %d %d %d %d %d\n",
						item.ResourceType,
						item.Count,
						item.Source["config"],
						item.Source["terraform"],
						item.Source["cloudformation"],
						item.Source["beanstalk"],
						item.Source["eks"],
						item.Source["both"],
						item.Source["config-only"],
					)
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
	cmd.Flags().BoolVar(&descending, "descending", false, "sort by descending instead of ascending; for by-type only")
	cmd.Flags().StringVar(&sortBy, "sort", sortByDefault, "sort order for results, options are: "+strings.Join(sortOptions, " ")+"; for by-type only")
	cmd.Flags().IntVar(&top, "top", 0, "limit to the top x results, use 0 for all, negative for last; for by-type only")
	return cmd
}
