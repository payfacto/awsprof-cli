package cmd

import (
	"fmt"
	"os"

	"github.com/payfacto/awsprof-cli/internal/shell"
	"github.com/spf13/cobra"
)

var shellInitCmd = &cobra.Command{
	Use:   "shell-init <bash|zsh|fish|powershell>",
	Short: "Print the shell hook to add to your shell profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := renderShellInit(args[0])
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, out)
		return nil
	},
}

func renderShellInit(name string) (string, error) {
	sh, err := shell.Parse(name)
	if err != nil {
		return "", err
	}
	return sh.Hook(), nil
}

func init() { rootCmd.AddCommand(shellInitCmd) }
