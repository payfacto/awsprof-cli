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
		if listPlain {
			fmt.Fprint(os.Stdout, renderPlainList(ps))
			return nil
		}
		r := lipgloss.NewRenderer(os.Stdout)
		fmt.Fprint(os.Stdout, renderList(ps, os.Getenv("AWS_PROFILE"), r))
		return nil
	},
}

// renderPlainList emits bare profile names, one per line, with no marker or
// color - the script-friendly `--plain` form.
func renderPlainList(ps []profiles.Profile) string {
	var b strings.Builder
	for _, p := range ps {
		fmt.Fprintf(&b, "%s\n", p.Name)
	}
	return b.String()
}

// renderList formats the human profile list: each name colored by its
// environment through r (which decides whether escapes are actually emitted),
// with the active profile marked " *".
func renderList(ps []profiles.Profile, active string, r *lipgloss.Renderer) string {
	var b strings.Builder
	for _, p := range ps {
		mark := ""
		if p.Name == active {
			mark = " *"
		}
		fmt.Fprintf(&b, "%s%s\n", envcolor.Render(p.Name, r), mark)
	}
	return b.String()
}
