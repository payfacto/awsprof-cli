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

func TestRenderList(t *testing.T) {
	ps := []profiles.Profile{{Name: "alpha"}, {Name: "beta"}}
	r := testRenderer(termenv.Ascii)

	plain := renderList(ps, "beta", true, r)
	if plain != "alpha\nbeta\n" {
		t.Fatalf("plain = %q", plain)
	}

	human := renderList(ps, "beta", false, r)
	if !strings.Contains(human, "beta *") {
		t.Fatalf("active profile not marked: %q", human)
	}
	if strings.Contains(human, "alpha *") {
		t.Fatalf("non-active profile wrongly marked: %q", human)
	}
}

func TestRenderListColorsEnvSegment(t *testing.T) {
	ps := []profiles.Profile{{Name: "payfacto-titan-prod-readonly"}}
	r := testRenderer(termenv.TrueColor)

	human := renderList(ps, "", false, r)
	if !strings.Contains(human, "\x1b[") {
		t.Fatalf("expected env color escapes, got %q", human)
	}

	// --plain must never colorize, even with a color-capable renderer.
	plain := renderList(ps, "", true, r)
	if strings.Contains(plain, "\x1b[") {
		t.Fatalf("--plain must stay byte-clean, got %q", plain)
	}
}
