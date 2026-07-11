// Package envcolor detects the deployment environment encoded in an AWS profile
// name and renders the name with only that segment colored.
package envcolor

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Env is the deployment environment inferred from a profile name.
type Env int

const (
	EnvNone Env = iota
	EnvProd
	EnvStaging
	EnvUAT
	EnvQA
	EnvDev
	EnvSandbox
)

// aliases maps a lowercased profile-name segment to its environment. Matching
// is against whole hyphen-delimited segments only, never substrings.
var aliases = map[string]Env{
	"prod":        EnvProd,
	"production":  EnvProd,
	"staging":     EnvStaging,
	"stage":       EnvStaging,
	"stg":         EnvStaging,
	"uat":         EnvUAT,
	"qa":          EnvQA,
	"dev":         EnvDev,
	"development": EnvDev,
	"sandbox":     EnvSandbox,
	"test":        EnvSandbox,
	"sbx":         EnvSandbox,
}

// colors is the per-environment foreground color. prod is additionally bold
// (applied in Render) as the one environment that must be unmistakable.
var colors = map[Env]lipgloss.Color{
	EnvProd:    lipgloss.Color("#ff5c57"),
	EnvStaging: lipgloss.Color("#ff9f43"),
	EnvUAT:     lipgloss.Color("#c586e0"),
	EnvQA:      lipgloss.Color("#f2cc60"),
	EnvDev:     lipgloss.Color("#57ab5a"),
	EnvSandbox: lipgloss.Color("#54aeff"),
}

// Detect returns the environment for a profile name and the index of the
// hyphen-delimited segment that matched. The first matching segment wins. A
// name with no recognized segment yields (EnvNone, -1).
func Detect(name string) (Env, int) {
	for i, seg := range strings.Split(name, "-") {
		if env, ok := aliases[strings.ToLower(seg)]; ok {
			return env, i
		}
	}
	return EnvNone, -1
}

// Render returns name with only its environment segment colored (prod also
// bold), styled through r. A name with no environment is returned unchanged.
// Whether escape codes are emitted is decided by the renderer's color profile,
// so callers may invoke Render unconditionally.
func Render(name string, r *lipgloss.Renderer) string {
	env, idx := Detect(name)
	if env == EnvNone {
		return name
	}
	segs := strings.Split(name, "-")
	style := r.NewStyle().Foreground(colors[env])
	if env == EnvProd {
		style = style.Bold(true)
	}
	segs[idx] = style.Render(segs[idx])
	return strings.Join(segs, "-")
}
