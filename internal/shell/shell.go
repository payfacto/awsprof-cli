// Package shell renders per-shell export statements and the shell-init hook.
package shell

import (
	"fmt"
	"strings"
)

// Shell is a supported target shell.
type Shell int

const (
	Bash Shell = iota
	Zsh
	Fish
	PowerShell
)

// Parse maps a shell name to a Shell.
func Parse(s string) (Shell, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "bash":
		return Bash, nil
	case "zsh":
		return Zsh, nil
	case "fish":
		return Fish, nil
	case "powershell", "pwsh":
		return PowerShell, nil
	default:
		return 0, fmt.Errorf("unsupported shell %q (want bash|zsh|fish|powershell)", s)
	}
}

// ExportLine returns the statement that sets AWS_PROFILE for this shell.
func (sh Shell) ExportLine(profile string) string {
	switch sh {
	case Fish:
		return fmt.Sprintf("set -gx AWS_PROFILE '%s'", profile)
	case PowerShell:
		return fmt.Sprintf("$env:AWS_PROFILE = %q", profile)
	default:
		return fmt.Sprintf("export AWS_PROFILE='%s'", profile)
	}
}

// Hook returns the shell wrapper printed by `awsprof shell-init <shell>`.
// The wrapper eval's only activation output; data commands pass through so
// their stdout is preserved.
func (sh Shell) Hook() string {
	switch sh {
	case Fish:
		return `function awsprof
    switch $argv[1]
        case list whoami shell-init completion help -h --help -v --version
            command awsprof $argv
        case '*'
            set -l out (command awsprof --shell fish $argv)
            or return
            test -n "$out"; and eval "$out"
    end
end`
	case PowerShell:
		return `function awsprof {
    switch ($args[0]) {
        {$_ -in 'list','whoami','shell-init','completion','help','-h','--help','-v','--version'} {
            & (Get-Command -CommandType Application awsprof).Source @args
        }
        default {
            $out = & (Get-Command -CommandType Application awsprof).Source --shell powershell @args
            if ($LASTEXITCODE -eq 0 -and $out) { Invoke-Expression ($out -join "` + "`n" + `") }
        }
    }
}`
	default: // Bash and Zsh share POSIX syntax.
		return `awsprof() {
  case "$1" in
    list|whoami|shell-init|completion|help|-h|--help|-v|--version)
      command awsprof "$@" ;;
    *)
      local out
      out="$(command awsprof --shell ` + sh.name() + ` "$@")" || return
      [ -n "$out" ] && eval "$out" ;;
  esac
}`
	}
}

func (sh Shell) name() string {
	switch sh {
	case Zsh:
		return "zsh"
	case Fish:
		return "fish"
	case PowerShell:
		return "powershell"
	default:
		return "bash"
	}
}
