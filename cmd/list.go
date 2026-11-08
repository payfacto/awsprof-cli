package cmd

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/payfacto/awsprof-cli/internal/envcolor"
	"github.com/payfacto/awsprof-cli/internal/profiles"
	"github.com/spf13/cobra"
)

var listPlain bool

const (
	activeMark     = " *"
	aliasColumnGap = 2
)

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
		resolver, err := loadResolver()
		if err != nil {
			return err
		}
		aliases, broken := profiles.AliasesByProfile(resolver, profileNames(ps))
		for _, aliasErr := range broken {
			fmt.Fprintf(os.Stderr, "warning: %v\n", aliasErr)
		}
		r := lipgloss.NewRenderer(os.Stdout)
		fmt.Fprint(os.Stdout, renderList(ps, listMarks{active: os.Getenv("AWS_PROFILE"), aliases: aliases}, r))
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

// listMarks is the per-profile annotation shown by the human list: which
// profile is active and the aliases (sorted) that resolve to each profile.
type listMarks struct {
	active  string
	aliases map[string][]string
}

// renderList formats the human profile list: each name colored by its
// environment through r (which decides whether escapes are actually emitted),
// with the active profile marked " *" and any aliases shown in an aligned
// trailing column.
func renderList(ps []profiles.Profile, marks listMarks, r *lipgloss.Renderer) string {
	labels := make([]string, len(ps))
	width := 0
	for i, p := range ps {
		labels[i] = p.Name
		if p.Name == marks.active {
			labels[i] += activeMark
		}
		width = max(width, utf8.RuneCountInString(labels[i]))
	}
	var b strings.Builder
	for i, p := range ps {
		b.WriteString(envcolor.Render(p.Name, r))
		b.WriteString(labels[i][len(p.Name):])
		if names := marks.aliases[p.Name]; len(names) > 0 {
			padding := strings.Repeat(" ", width-utf8.RuneCountInString(labels[i])+aliasColumnGap)
			b.WriteString(padding + strings.Join(names, ", "))
		}
		b.WriteString("\n")
	}
	return b.String()
}
