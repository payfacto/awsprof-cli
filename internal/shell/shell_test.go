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
		PowerShell: "$env:AWS_PROFILE = \"dev\"",
	}
	for sh, want := range cases {
		if got := sh.ExportLine("dev"); got != want {
			t.Errorf("ExportLine(%v) = %q, want %q", sh, got, want)
		}
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
