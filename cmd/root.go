// Package cmd wires the awsprof command tree.
package cmd

import (
	"github.com/spf13/cobra"
)

// Version is the CLI version. It defaults to "dev" for plain `go build` and is
// overridden at release time via
// -ldflags -X 'github.com/payfacto/awsprof-cli/cmd.Version=...'.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:           "awsprof [profile]",
	Short:         "Pick an AWS profile to log in as",
	Args:          cobra.MaximumNArgs(1),
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Filled in Task 11 (picker when no args; activate when one arg).
		return cmd.Help()
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
