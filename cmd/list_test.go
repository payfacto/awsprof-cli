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
	human := renderList(ps, listMarks{active: "beta"}, testRenderer(termenv.Ascii))
	if !strings.Contains(human, "beta *") {
		t.Fatalf("active profile not marked: %q", human)
	}
	if strings.Contains(human, "alpha *") {
		t.Fatalf("non-active profile wrongly marked: %q", human)
	}
}

func TestRenderListColorsEnvSegment(t *testing.T) {
	ps := []profiles.Profile{{Name: "payfacto-titan-prod-readonly"}}
	human := renderList(ps, listMarks{}, testRenderer(termenv.TrueColor))
	if !strings.Contains(human, "\x1b[") {
		t.Fatalf("expected env color escapes, got %q", human)
	}
}

func TestRenderListAliasColumn(t *testing.T) {
	ps := []profiles.Profile{{Name: "payfacto-synapse-admin"}, {Name: "default"}, {Name: "payfacto-sandbox-admin"}}
	aliases := map[string][]string{
		"payfacto-synapse-admin": {"sa", "syn"},
		"payfacto-sandbox-admin": {"ba"},
	}
	got := renderList(ps, listMarks{active: "default", aliases: aliases}, testRenderer(termenv.Ascii))
	want := "" +
		"payfacto-synapse-admin  sa, syn\n" +
		"default *\n" +
		"payfacto-sandbox-admin  ba\n"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestRenderListAliasColumnAlignsWithActiveMark(t *testing.T) {
	ps := []profiles.Profile{{Name: "abc"}, {Name: "a"}}
	aliases := map[string][]string{"abc": {"x"}, "a": {"y"}}
	got := renderList(ps, listMarks{active: "a", aliases: aliases}, testRenderer(termenv.Ascii))
	want := "abc  x\na *  y\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRenderListAliasColumnCountsRunes(t *testing.T) {
	ps := []profiles.Profile{{Name: "café"}, {Name: "abcde"}}
	aliases := map[string][]string{"café": {"c"}, "abcde": {"a"}}
	got := renderList(ps, listMarks{aliases: aliases}, testRenderer(termenv.Ascii))
	want := "café   c\nabcde  a\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
