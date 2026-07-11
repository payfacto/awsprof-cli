// Package cmd wires the awsprof command tree.
package cmd

import (
	"errors"
	"os"

	"github.com/payfacto/awsprof-cli/internal/picker"
	"github.com/payfacto/awsprof-cli/internal/profiles"
	"github.com/payfacto/awsprof-cli/internal/shell"
	"github.com/spf13/cobra"
)

// Version is the CLI version. It defaults to "dev" for plain `go build` and is
// overridden at release time via
// -ldflags -X 'github.com/payfacto/awsprof-cli/cmd.Version=...'.
var Version = "dev"

// shellFlag is the target shell for export syntax, set via the persistent
// --shell flag and resolved through resolveShell().
var shellFlag string

var rootCmd = &cobra.Command{
	Use:           "awsprof [profile]",
	Short:         "Pick an AWS profile to log in as",
	Args:          cobra.MaximumNArgs(1),
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		sh := resolveShell()
		if len(args) == 1 {
			return activate(cmd.Context(), args[0], sh)
		}

		ps, err := profiles.List()
		if err != nil {
			return err
		}
		if len(ps) == 0 {
			return errors.New("no AWS profiles found (checked ~/.aws/config and ~/.aws/credentials)")
		}

		choice, err := picker.Pick(picker.BuildItems(ps, os.Getenv("AWS_PROFILE")))
		if err != nil {
			return err
		}
		if choice == "" {
			// User aborted the picker; exit quietly with no export.
			return nil
		}
		return activate(cmd.Context(), choice, sh)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&shellFlag, "shell", "bash", "target shell for export syntax (bash|zsh|fish|powershell)")
}

// resolveShell parses --shell, falling back to bash on an unrecognized value.
func resolveShell() shell.Shell {
	sh, err := shell.Parse(shellFlag)
	if err != nil {
		return shell.Bash
	}
	return sh
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
