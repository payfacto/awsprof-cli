package cmd

import "testing"

func TestEffectiveVersion(t *testing.T) {
	tests := []struct {
		name      string
		ldflags   string
		buildInfo string
		want      string
	}{
		{"ldflags value wins", "v0.1.0", "v9.9.9", "v0.1.0"},
		{"ldflags wins over devel build info", "v0.1.0", "(devel)", "v0.1.0"},
		{"go install falls back to build info", "dev", "v0.1.1", "v0.1.1"},
		{"devel build info ignored", "dev", "(devel)", "dev"},
		{"empty build info ignored", "dev", "", "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := effectiveVersion(tt.ldflags, tt.buildInfo); got != tt.want {
				t.Errorf("effectiveVersion(%q, %q) = %q, want %q",
					tt.ldflags, tt.buildInfo, got, tt.want)
			}
		})
	}
}
