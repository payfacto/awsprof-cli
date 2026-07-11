package cmd

import (
	"regexp"
	"strings"
	"testing"

	"github.com/muesli/termenv"
	"github.com/payfacto/awsprof-cli/internal/identity"
)

var cmdAnsiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

func TestWhoamiLineUnsetShowsDefaultAndIsPlain(t *testing.T) {
	id := identity.Identity{Account: "123", Arn: "arn"}
	got := whoamiLine("", id, testRenderer(termenv.Ascii))
	if strings.Contains(got, "\x1b") {
		t.Fatalf("no-color unset line must be byte-clean, got %q", got)
	}
	want := "AWS_PROFILE=default (unset)\nAccount 123\narn\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWhoamiLineSetProfilePlain(t *testing.T) {
	id := identity.Identity{Account: "1", Arn: "arn"}
	got := whoamiLine("work", id, testRenderer(termenv.Ascii))
	if !strings.HasPrefix(got, "AWS_PROFILE=work\n") {
		t.Errorf("got %q", got)
	}
}

func TestWhoamiLineColorsEnvSegment(t *testing.T) {
	id := identity.Identity{Account: "1", Arn: "arn"}
	got := whoamiLine("payfacto-titan-prod-readonly", id, testRenderer(termenv.TrueColor))
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected env color escapes, got %q", got)
	}
	stripped := cmdAnsiRE.ReplaceAllString(got, "")
	if !strings.HasPrefix(stripped, "AWS_PROFILE=payfacto-titan-prod-readonly\n") {
		t.Errorf("stripped = %q", stripped)
	}
}
