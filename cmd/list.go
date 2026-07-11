package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/payfacto/awsprof-cli/internal/envcolor"
	"github.com/payfacto/awsprof-cli/internal/profiles"
	"github.com/spf13/cobra"
)

var listPlain bool

func init() {
	listCmd.Flags().BoolVar(&listPlain, "plain", false, "print bare profile names only")
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available AWS profiles",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ps, err := profiles.List()
		if err != nil {
			return err
		}
		r := lipgloss.NewRenderer(os.Stdout)
		fmt.Fprint(os.Stdout, renderList(ps, os.Getenv("AWS_PROFILE"), listPlain, r))
		return nil
	},
}

// renderList formats the profile list. --plain (plain=true) emits bare names
// with no marker or color for scripting. Otherwise each name is colored by its
// environment through r (which decides whether escapes are actually emitted),
// and the active profile is marked with " *".
func renderList(ps []profiles.Profile, active string, plain bool, r *lipgloss.Renderer) string {
	var b strings.Builder
	for _, p := range ps {
		if plain {
			fmt.Fprintf(&b, "%s\n", p.Name)
			continue
		}
		mark := ""
		if p.Name == active {
			mark = " *"
		}
		fmt.Fprintf(&b, "%s%s\n", envcolor.Render(p.Name, r), mark)
	}
	return b.String()
}
