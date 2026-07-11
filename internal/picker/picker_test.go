package picker

import (
	"testing"

	"github.com/payfacto/awsprof-cli/internal/profiles"
)

func TestSelectHeight(t *testing.T) {
	tests := []struct {
		name       string
		termHeight int
		itemCount  int
		want       int
	}{
		{"unknown size falls back", 0, 50, 10},
		{"negative size falls back", -1, 50, 10},
		{"fallback clamped to item count", 0, 3, 5},
		{"bounded by terminal height", 20, 50, 15},
		{"clamped to item count", 50, 5, 7},
		{"floored at minimum", 6, 50, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := selectHeight(tt.termHeight, tt.itemCount); got != tt.want {
				t.Errorf("selectHeight(%d, %d) = %d, want %d",
					tt.termHeight, tt.itemCount, got, tt.want)
			}
		})
	}
}

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
