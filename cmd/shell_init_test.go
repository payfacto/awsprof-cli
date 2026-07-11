package cmd

import (
	"strings"
	"testing"
)

func TestShellInitOutput(t *testing.T) {
	out, err := renderShellInit("bash")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "awsprof()") {
		t.Fatalf("bash hook missing: %q", out)
	}
	if _, err := renderShellInit("tcsh"); err == nil {
		t.Fatal("expected error for unsupported shell")
	}
}
