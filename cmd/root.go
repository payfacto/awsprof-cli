// Package cmd wires the awsprof command tree.
package cmd

import (
	"errors"
	"os"
	"runtime/debug"

	"github.com/payfacto/awsprof-cli/internal/picker"
	"github.com/payfacto/awsprof-cli/internal/profiles"
	"github.com/payfacto/awsprof-cli/internal/shell"
	"github.com/spf13/cobra"
)

// Version is the CLI version. It defaults to "dev" for plain `go build` and is
// overridden at release time via
// -ldflags -X 'github.com/payfacto/awsprof-cli/cmd.Version=...'.
var Version = "dev"

// effectiveVersion resolves the version to report. An explicit ldflags value
// (anything other than the "dev" default) always wins. Otherwise - as with a
// binary from `go install <module>/cmd/awsprof@<version>`, which does not run
// our ldflags - it falls back to the module version embedded in the build info,
// so those binaries report the real tag instead of "dev".
func effectiveVersion(ldflags, buildInfo string) string {
	if ldflags != "dev" {
		return ldflags
	}
	if buildInfo != "" && buildInfo != "(devel)" {
		return buildInfo
	}
	return ldflags
}

// mainModuleVersion returns the main module's version from the embedded build
// info, or "" when it is unavailable.
func mainModuleVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}
	return ""
}

// shellFlag is the target shell for export syntax, set via the persistent
// --shell flag and resolved through resolveShell().
var shellFlag string

var rootCmd = &cobra.Command{
	Use:           "awsprof [profile]",
	Short:         "Pick an AWS profile to log in as",
	Args:          cobra.MaximumNArgs(1),
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
			if errors.Is(err, picker.ErrAborted) {
				return nil // user cancelled the picker; nothing to activate
			}
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
	rootCmd.Version = effectiveVersion(Version, mainModuleVersion())
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
