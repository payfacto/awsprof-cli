package cmd

import (
	"strings"
	"testing"

	"github.com/payfacto/awsprof-cli/internal/profiles"
)

func TestRenderList(t *testing.T) {
	ps := []profiles.Profile{{Name: "alpha"}, {Name: "beta"}}

	plain := renderList(ps, "beta", true)
	if plain != "alpha\nbeta\n" {
		t.Fatalf("plain = %q", plain)
	}

	human := renderList(ps, "beta", false)
	if !strings.Contains(human, "beta *") {
		t.Fatalf("active profile not marked: %q", human)
	}
	if strings.Contains(human, "alpha *") {
		t.Fatalf("non-active profile wrongly marked: %q", human)
	}
}
