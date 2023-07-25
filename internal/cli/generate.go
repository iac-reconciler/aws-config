package cli

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/iac-reconciler/tf-aws-config/pkg/compare"
	"github.com/iac-reconciler/tf-aws-config/pkg/load"
	"github.com/spf13/cobra"
)

func generate() *cobra.Command {
	var (
		tfRecursive, doSummary bool
	)

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
			reconciled, err := compare.Reconcile(snapshot, tfstates)
			if err != nil {
				return fmt.Errorf("unable to reconcile: %w", err)
			}
			fmt.Printf("Both: %d\n", len(reconciled.Both))
			fmt.Printf("Terraform Only: %d\n", len(reconciled.TerraformOnly))
			fmt.Printf("Config Only: %d\n", len(reconciled.ConfigOnly))
			fmt.Printf("Terraform Files: %d\n", reconciled.TerraformFiles)

			// no error
			return nil
		},
	}

	cmd.Flags().BoolVar(&tfRecursive, "tf-recursive", false, "treat the path to terraform state as a directory and recursively search for .tfstate files")
	cmd.Flags().BoolVar(&doSummary, "summary", false, "provide summary results")
	return cmd
}
