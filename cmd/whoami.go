package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/payfacto/awsprof-cli/internal/envcolor"
	"github.com/payfacto/awsprof-cli/internal/identity"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the current AWS identity without switching",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		profile := os.Getenv("AWS_PROFILE")
		id, err := identity.Check(cmd.Context(), profile)
		if err != nil {
			label := profile
			if label == "" {
				label = "default (AWS_PROFILE unset)"
			}
			return fmt.Errorf("not authenticated for %s: %w", label, err)
		}
		r := lipgloss.NewRenderer(os.Stdout)
		fmt.Fprint(os.Stdout, whoamiLine(profile, id, r))
		return nil
	},
}

// whoamiLine formats the whoami output. When AWS_PROFILE is unset the SDK uses
// the `default` profile, so the effective name is shown as "default" with a
// dim "(unset)" hint; when set, the profile name is shown as-is. The name is
// colored by its environment through r.
func whoamiLine(profile string, id identity.Identity, r *lipgloss.Renderer) string {
	name := profile
	suffix := ""
	if profile == "" {
		name = "default"
		suffix = r.NewStyle().Faint(true).Render(" (unset)")
	}
	return fmt.Sprintf("AWS_PROFILE=%s%s\nAccount %s\n%s\n",
		envcolor.Render(name, r), suffix, id.Account, id.Arn)
}

func init() { rootCmd.AddCommand(whoamiCmd) }
