package cmd

import (
	"fmt"
	"os"
	"strings"

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
		fmt.Fprint(os.Stdout, renderList(ps, os.Getenv("AWS_PROFILE"), listPlain))
		return nil
	},
}

func renderList(ps []profiles.Profile, active string, plain bool) string {
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
		fmt.Fprintf(&b, "%s%s\n", p.Name, mark)
	}
	return b.String()
}
