package shell

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := map[string]Shell{"bash": Bash, "ZSH": Zsh, "fish": Fish, "powershell": PowerShell, "pwsh": PowerShell}
	for in, want := range cases {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %v, %v", in, got, err)
		}
	}
	if _, err := Parse("tcsh"); err == nil {
		t.Errorf("Parse(tcsh): expected error")
	}
}

func TestExportLine(t *testing.T) {
	cases := map[Shell]string{
		Bash:       "export AWS_PROFILE='dev'",
		Zsh:        "export AWS_PROFILE='dev'",
		Fish:       "set -gx AWS_PROFILE 'dev'",
		PowerShell: "$env:AWS_PROFILE = 'dev'",
	}
	for sh, want := range cases {
		if got := sh.ExportLine("dev"); got != want {
			t.Errorf("ExportLine(%v) = %q, want %q", sh, got, want)
		}
	}
}

// A crafted profile name must not break out of its quotes or interpolate.
func TestExportLine_Escaping(t *testing.T) {
	cases := []struct {
		name    string
		sh      Shell
		profile string
		want    string
	}{
		{"posix single quote", Bash, "a'b", `export AWS_PROFILE='a'\''b'`},
		{"posix injection attempt", Zsh, "x'; rm -rf ~; '", `export AWS_PROFILE='x'\''; rm -rf ~; '\'''`},
		{"powershell single quote", PowerShell, "a'b", "$env:AWS_PROFILE = 'a''b'"},
		{"powershell dollar not interpolated", PowerShell, "payfacto$x", "$env:AWS_PROFILE = 'payfacto$x'"},
		{"powershell subexpr not executed", PowerShell, "$(calc)", "$env:AWS_PROFILE = '$(calc)'"},
		{"fish single quote", Fish, "a'b", `set -gx AWS_PROFILE 'a\'b'`},
		{"fish backslash", Fish, `a\b`, `set -gx AWS_PROFILE 'a\\b'`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.sh.ExportLine(c.profile); got != c.want {
				t.Errorf("ExportLine(%v, %q) = %q, want %q", c.sh, c.profile, got, c.want)
			}
		})
	}
}

func TestHook_ContainsWrapperAndPassthrough(t *testing.T) {
	h := Bash.Hook()
	if !strings.Contains(h, "awsprof()") {
		t.Errorf("bash hook missing function definition")
	}
	if !strings.Contains(h, "command awsprof") {
		t.Errorf("bash hook missing passthrough to real binary")
	}
	if !strings.Contains(h, "eval") {
		t.Errorf("bash hook missing eval of activation output")
	}
	if !strings.Contains(Fish.Hook(), "function awsprof") {
		t.Errorf("fish hook missing function")
	}
	if !strings.Contains(PowerShell.Hook(), "function awsprof") {
		t.Errorf("powershell hook missing function")
	}
}

func TestHook_EmbedsShellName(t *testing.T) {
	if !strings.Contains(Bash.Hook(), "--shell bash") {
		t.Errorf("bash hook must embed --shell bash")
	}
	if !strings.Contains(Zsh.Hook(), "--shell zsh") {
		t.Errorf("zsh hook must embed --shell zsh")
	}
}
