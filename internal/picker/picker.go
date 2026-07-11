// Package picker provides the interactive profile selector.
package picker

import (
	"os"

	"github.com/charmbracelet/huh"
	"github.com/payfacto/awsprof-cli/internal/profiles"
)

// ErrAborted is returned by Pick when the user cancels the selection (Esc/Ctrl-C).
var ErrAborted = huh.ErrUserAborted

// Item is one selectable profile row.
type Item struct {
	Label string
	Value string
}

// BuildItems turns profiles into picker items, marking the active one.
func BuildItems(ps []profiles.Profile, active string) []Item {
	items := make([]Item, 0, len(ps))
	for _, p := range ps {
		label := p.Name
		if p.Name == active {
			label = p.Name + " (active)"
		}
		items = append(items, Item{Label: label, Value: p.Name})
	}
	return items
}

// Pick shows a filterable single-select and returns the chosen profile name.
// The UI renders to stderr so stdout stays reserved for the export line.
func Pick(items []Item) (string, error) {
	opts := make([]huh.Option[string], len(items))
	for i, it := range items {
		opts[i] = huh.NewOption(it.Label, it.Value)
	}
	var selected string
	field := huh.NewSelect[string]().
		Title("Select an AWS profile").
		Options(opts...).
		Filtering(true).
		Value(&selected)
	err := huh.NewForm(huh.NewGroup(field)).WithOutput(os.Stderr).Run()
	return selected, err
}
