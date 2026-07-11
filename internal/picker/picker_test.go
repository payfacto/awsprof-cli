package picker

import (
	"testing"

	"github.com/payfacto/awsprof-cli/internal/profiles"
)

func TestBuildItems(t *testing.T) {
	ps := []profiles.Profile{{Name: "alpha"}, {Name: "beta"}}
	items := BuildItems(ps, "beta")
	if len(items) != 2 {
		t.Fatalf("got %d items", len(items))
	}
	if items[0].Value != "alpha" || items[0].Label != "alpha" {
		t.Fatalf("alpha item wrong: %+v", items[0])
	}
	if items[1].Value != "beta" || items[1].Label != "beta (active)" {
		t.Fatalf("active label wrong: %+v", items[1])
	}
}
