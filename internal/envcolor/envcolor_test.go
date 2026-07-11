package envcolor

import (
	"io"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

// sgrParams returns the ";"-separated parameters of every SGR escape in s.
func sgrParams(s string) []string {
	var params []string
	for _, m := range ansiRE.FindAllString(s, -1) {
		body := strings.TrimSuffix(strings.TrimPrefix(m, "\x1b["), "m")
		params = append(params, strings.Split(body, ";")...)
	}
	return params
}

func renderer(p termenv.Profile) *lipgloss.Renderer {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(p)
	return r
}

func TestDetect(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantEnv Env
		wantIdx int
	}{
		{"prod", "payfacto-titan-prod-readonly", EnvProd, 2},
		{"production alias", "payfacto-production-admin", EnvProd, 1},
		{"staging", "payfacto-gateway-staging-poweruser", EnvStaging, 2},
		{"stage alias", "acme-stage-ro", EnvStaging, 1},
		{"stg alias", "acme-stg-ro", EnvStaging, 1},
		{"uat", "payfacto-titan-uat-readonly", EnvUAT, 2},
		{"qa", "payfacto-titan-qa-poweruser", EnvQA, 2},
		{"dev", "payfacto-gateway-dev-readonly", EnvDev, 2},
		{"development alias", "payfacto-development-ro", EnvDev, 1},
		{"sandbox", "payfacto-sandbox-admin", EnvSandbox, 1},
		{"test alias to sandbox", "payfacto-test-ro", EnvSandbox, 1},
		{"sbx alias to sandbox", "payfacto-sbx-ro", EnvSandbox, 1},
		{"case-insensitive", "payfacto-PROD-readonly", EnvProd, 1},
		{"no env", "payfacto-synapse-readonly", EnvNone, -1},
		{"substring devops not dev", "payfacto-devops-tools", EnvNone, -1},
		{"substring nonprod not prod", "payfacto-nonprod-x", EnvNone, -1},
		{"first match wins", "payfacto-dev-prod-x", EnvDev, 1},
		{"empty", "", EnvNone, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEnv, gotIdx := Detect(tt.input)
			if gotEnv != tt.wantEnv || gotIdx != tt.wantIdx {
				t.Errorf("Detect(%q) = (%v, %d), want (%v, %d)", tt.input, gotEnv, gotIdx, tt.wantEnv, tt.wantIdx)
			}
		})
	}
}

func TestRenderColorsOnlyEnvSegment(t *testing.T) {
	const name = "payfacto-titan-prod-readonly"
	out := Render(name, renderer(termenv.TrueColor))

	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI escapes, got %q", out)
	}
	if got := stripANSI(out); got != name {
		t.Errorf("stripped output = %q, want original %q", got, name)
	}
	if !strings.HasPrefix(out, "payfacto-titan-") {
		t.Errorf("segments before env should be uncolored, got %q", out)
	}
	if !strings.HasSuffix(out, "-readonly") {
		t.Errorf("segments after env should be uncolored, got %q", out)
	}
}

func TestRenderProdIsBold(t *testing.T) {
	out := Render("acme-prod-ro", renderer(termenv.TrueColor))
	if !slices.Contains(sgrParams(out), "1") {
		t.Errorf("prod should render bold (SGR param 1), got %q", out)
	}
}

func TestRenderNonProdIsNotBold(t *testing.T) {
	out := Render("acme-dev-ro", renderer(termenv.TrueColor))
	if slices.Contains(sgrParams(out), "1") {
		t.Errorf("non-prod env should not be bold, got %q", out)
	}
}

func TestRenderNoEnvUnchanged(t *testing.T) {
	const name = "payfacto-synapse-readonly"
	for _, p := range []termenv.Profile{termenv.TrueColor, termenv.Ascii} {
		if out := Render(name, renderer(p)); out != name {
			t.Errorf("no-env name should be unchanged, got %q", out)
		}
	}
}

func TestRenderAsciiProfileIsPlain(t *testing.T) {
	const name = "payfacto-gateway-dev-readonly"
	out := Render(name, renderer(termenv.Ascii))
	if out != name {
		t.Errorf("ascii profile should emit no escapes, got %q", out)
	}
}
