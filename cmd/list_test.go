package cmd

import (
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/payfacto/awsprof-cli/internal/profiles"
)

func testRenderer(p termenv.Profile) *lipgloss.Renderer {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(p)
	return r
}

func TestRenderPlainList(t *testing.T) {
	ps := []profiles.Profile{{Name: "alpha"}, {Name: "payfacto-titan-prod-readonly"}}

	plain := renderPlainList(ps)
	if plain != "alpha\npayfacto-titan-prod-readonly\n" {
		t.Fatalf("plain = %q", plain)
	}
	// --plain must never colorize.
	if strings.Contains(plain, "\x1b[") {
		t.Fatalf("--plain must stay byte-clean, got %q", plain)
	}
}

func TestRenderList(t *testing.T) {
	ps := []profiles.Profile{{Name: "alpha"}, {Name: "beta"}}
	human := renderList(ps, "beta", testRenderer(termenv.Ascii))
	if !strings.Contains(human, "beta *") {
		t.Fatalf("active profile not marked: %q", human)
	}
	if strings.Contains(human, "alpha *") {
		t.Fatalf("non-active profile wrongly marked: %q", human)
	}
}

func TestRenderListColorsEnvSegment(t *testing.T) {
	ps := []profiles.Profile{{Name: "payfacto-titan-prod-readonly"}}
	human := renderList(ps, "", testRenderer(termenv.TrueColor))
	if !strings.Contains(human, "\x1b[") {
		t.Fatalf("expected env color escapes, got %q", human)
	}
}
