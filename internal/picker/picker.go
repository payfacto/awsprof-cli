// Package picker provides the interactive profile selector.
package picker

import (
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/payfacto/awsprof-cli/internal/envcolor"
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

// selectHeight bounds the visible option list so the picker fits the terminal.
// Without an explicit height the select sizes its viewport to the full option
// count; on terminals that do not report their size (e.g. Git Bash / mintty)
// huh never receives a window-size message to clamp it, so the frame overflows
// and the screen scrolls instead of moving the selection cursor.
//
// termHeight <= 0 means the size is unknown and a conservative fallback is used.
// The returned height is never taller than the content itself (title,
// description, and options) nor shorter than a usable minimum.
func selectHeight(termHeight, itemCount int) int {
	const (
		reserve  = 5  // lines for the help footer and padding
		chrome   = 2  // the field's title and description lines
		minRows  = 4  // smallest usable field height
		fallback = 10 // used when the terminal size is unknown
	)
	rows := fallback
	if termHeight > 0 {
		rows = termHeight - reserve
	}
	if contentRows := itemCount + chrome; rows > contentRows {
		rows = contentRows
	}
	if rows < minRows {
		rows = minRows
	}
	return rows
}

// Pick shows a filterable single-select and returns the chosen profile name.
// The UI renders to stderr so stdout stays reserved for the export line.
func Pick(items []Item) (string, error) {
	// Bind the color renderer to stderr (where the picker renders) so color is
	// enabled/disabled by that stream's TTY + NO_COLOR state.
	r := lipgloss.NewRenderer(os.Stderr)
	opts := make([]huh.Option[string], len(items))
	for i, it := range items {
		opts[i] = huh.NewOption(envcolor.Render(it.Label, r), it.Value)
	}
	// A failure here means the size is unknown (non-TTY); selectHeight handles
	// the resulting zero height with its fallback.
	_, termHeight, _ := term.GetSize(os.Stderr.Fd())
	var selected string
	field := huh.NewSelect[string]().
		Title("Select an AWS profile").
		Description("ctrl+c to cancel").
		Options(opts...).
		Height(selectHeight(termHeight, len(items))).
		Value(&selected)
	err := huh.NewForm(huh.NewGroup(field)).
		WithOutput(os.Stderr).
		WithKeyMap(pickerKeyMap()).
		Run()
	return selected, err
}

// pickerKeyMap adjusts huh's default keymap for the single-field picker.
// There is only one field, so the field-to-field navigation bindings
// ("prev"/"next") are meaningless; disabling them keeps them out of the help
// footer regardless of huh's per-position setup. Submit's Enter is relabeled
// "select" to read naturally for a picker; Enter still submits via Submit.
func pickerKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()
	km.Select.Prev.SetEnabled(false)
	km.Select.Next.SetEnabled(false)
	km.Select.Submit.SetHelp("enter", "select")
	return km
}
