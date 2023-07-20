package cli

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"

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
			)
			if tfRecursive {
				tfstate, err = fs.Glob(os.DirFS(args[1]), "**/*.tfstate")
				if err != nil {
					return err
				}
			} else {
				tfstate = []string{args[1]}
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
			var tfstates []load.TerraformState
			for _, tfstateFile := range tfstate {
				f, err := os.Open(tfstateFile)
				if err != nil {
					return fmt.Errorf("unable to open tfstate file %s: %w", tfstateFile, err)
				}
				defer f.Close()
				var state load.TerraformState
				if err := json.NewDecoder(f).Decode(&state); err != nil {
					return fmt.Errorf("unable to decode terraform state file %s: %w", tfstateFile, err)
				}
				tfstates = append(tfstates, state)
			}
			// all loaded, now run the reconcile
			reconciled, err := compare.Reconcile(snapshot, tfstates)
			if err != nil {
				return fmt.Errorf("unable to reconcile: %w", err)
			}
			fmt.Println(reconciled)
			if doSummary {
				summary, err := compare.Summarize(snapshot, tfstates)
				if err != nil {
					return fmt.Errorf("unable to generate summary: %w", err)
				}
				fmt.Println(summary)
			}

			// no error
			return nil
		},
	}

	cmd.Flags().BoolVar(&tfRecursive, "tf-recursive", false, "treat the path to terraform state as a directory and recursively search for .tfstate files")
	cmd.Flags().BoolVar(&doSummary, "summary", false, "provide summary results")
	return cmd
}
