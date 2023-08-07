package cli

import (
	"fmt"

	"github.com/iac-reconciler/aws-config/pkg/compare"
	"github.com/spf13/cobra"
)

func summarize() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "summarize",
		Short:   "Summarize the resources by source",
		Long:    `Summarize the resources by source.`,
		Example: `  aws-config summarize --aws-config <aws-config-snapshot.json> --terraform <terraform.tfstate>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, err := compare.Summarize(items)
			if err != nil {
				return fmt.Errorf("unable to summarize: %w", err)
			}

			fmt.Printf("Summary:\n")
			fmt.Printf("Both (Config+IaC): %d\n", summary.BothResources)
			fmt.Printf("Source All Only Mapped Unmapped\n")
			for _, source := range summary.Sources {
				fmt.Printf("%s: %d %d %d %d\n", source.Name, source.Total, source.OnlyCount, source.OnlyMappedCount, source.OnlyUnmappedCount)
			}
			fmt.Printf("Terraform Files: %d\n", len(tfstates))

			// no error
			return nil
		},
	}

	return cmd
}
