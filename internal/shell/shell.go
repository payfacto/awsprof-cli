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
		return 0, fmt.Errorf("unsupported shell %q (want bash|zsh|fish|powershell|pwsh)", s)
	}
}

// ExportLine returns the statement that sets AWS_PROFILE for this shell. The
// installed hook eval's / Invoke-Expression's this line, so the profile name is
// quoted per shell to prevent a crafted name (e.g. from a malicious
// ~/.aws/config section) from breaking out and executing code.
func (sh Shell) ExportLine(profile string) string {
	switch sh {
	case Fish:
		return "set -gx AWS_PROFILE " + fishSingleQuote(profile)
	case PowerShell:
		return "$env:AWS_PROFILE = " + powershellSingleQuote(profile)
	default:
		return "export AWS_PROFILE=" + posixSingleQuote(profile)
	}
}

// posixSingleQuote single-quotes s for bash/zsh, rendering an embedded single
// quote as the '\” idiom so the value cannot escape the quotes.
func posixSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// fishSingleQuote single-quotes s for fish, where inside single quotes only
// backslash and single quote are special (escaped with a backslash). Backslash
// is escaped first so the backslash added for a quote is not re-escaped.
func fishSingleQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "'", `\'`)
	return "'" + s + "'"
}

// powershellSingleQuote single-quotes s for PowerShell, doubling an embedded
// single quote. A single-quoted PowerShell string is literal - no $ or $(...)
// interpolation - so the value cannot inject when Invoke-Expression'd.
func powershellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
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
