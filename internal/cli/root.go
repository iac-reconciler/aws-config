package cli

import (
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{Use: "tf-aws-config"}
	verbose bool
)

func init() {
	rootCmd.AddCommand(generate())

	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "print lots of output to stderr")
}

// Execute primary function for cobra
func Execute() {
	_ = rootCmd.Execute()
}
