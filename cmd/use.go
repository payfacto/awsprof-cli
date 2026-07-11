package cmd

import "github.com/spf13/cobra"

// useCmd is a hidden alias of `awsprof <profile>`, kept for scripts/muscle
// memory that expect an explicit verb.
var useCmd = &cobra.Command{
	Use:    "use <profile>",
	Short:  "Activate a profile by name (hidden alias of `awsprof <profile>`)",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sh := resolveShell()
		return activate(cmd.Context(), args[0], sh)
	},
}

func init() { rootCmd.AddCommand(useCmd) }
