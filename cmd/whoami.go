package cmd

import (
	"fmt"
	"os"

	"github.com/payfacto/awsprof-cli/internal/identity"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the current AWS identity without switching",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		profile := os.Getenv("AWS_PROFILE")
		display := profile
		if display == "" {
			display = "(unset -> default)"
		}
		id, err := identity.Check(cmd.Context(), profile)
		if err != nil {
			return fmt.Errorf("not authenticated for %s: %w", display, err)
		}
		fmt.Fprintf(os.Stdout, "AWS_PROFILE=%s\nAccount %s\n%s\n", display, id.Account, id.Arn)
		return nil
	},
}

func init() { rootCmd.AddCommand(whoamiCmd) }
