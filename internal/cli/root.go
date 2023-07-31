package cli

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/iac-reconciler/aws-config/pkg/compare"
	"github.com/iac-reconciler/aws-config/pkg/load"
	"github.com/spf13/cobra"
)

var (
	rootCmd  = root()
	verbose  bool
	items    []*compare.LocatedItem
	tfstates = make(map[string]load.TerraformState)
)

func root() *cobra.Command {
	var (
		tfRecursive                 bool
		snapshotFile, terraformPath string
	)
	cmd := &cobra.Command{
		Use: "aws-config",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var (
				tfstate []string
				err     error
				fsys    fs.FS
			)
			if tfRecursive {
				fsys = os.DirFS(terraformPath)
				if err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
					if filepath.Ext(path) == ".tfstate" {
						tfstate = append(tfstate, path)
					}
					return nil
				}); err != nil {
					return err
				}
			} else {
				fsys = os.DirFS(path.Dir(terraformPath))
				tfstate = []string{path.Base(terraformPath)}
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
			items, err = compare.Reconcile(snapshot, tfstates)
			if err != nil {
				return fmt.Errorf("unable to reconcile: %w", err)
			}
			return nil
		},
	}
	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "print lots of output to stderr")
	cmd.PersistentFlags().BoolVar(&tfRecursive, "tf-recursive", false, "treat the path to terraform state as a directory and recursively search for .tfstate files")
	cmd.PersistentFlags().StringVar(&terraformPath, "terraform", "", "path to the terraform state file or directory containing .tfstate files; required")
	cmd.PersistentFlags().StringVar(&snapshotFile, "aws-config", "", "path to the AWS Config snapshot json file; required")
	_ = cmd.MarkPersistentFlagRequired("terraform")
	_ = cmd.MarkPersistentFlagRequired("aws-config")

	return cmd
}

func init() {
	rootCmd.AddCommand(summarize())
	rootCmd.AddCommand(detail())
	rootCmd.AddCommand(resources())
}

// Execute primary function for cobra
func Execute() {
	_ = rootCmd.Execute()
}
