package profiles

import "testing"

func TestResolve(t *testing.T) {
	names := []string{"payfacto-synapse-admin", "corp-data", "default"}
	cases := []struct {
		in       string
		prefixes []string
		want     string
		wantErr  bool
	}{
		{"payfacto-synapse-admin", []string{"payfacto-"}, "payfacto-synapse-admin", false},
		{"synapse-admin", []string{"payfacto-"}, "payfacto-synapse-admin", false},
		{"data", []string{"payfacto-", "corp-"}, "corp-data", false},
		{"nope", []string{"payfacto-"}, "", true},
		{"default", nil, "default", false},
	}
	for _, c := range cases {
		got, err := Resolve(c.in, c.prefixes, names)
		if c.wantErr {
			if err == nil {
				t.Errorf("Resolve(%q): expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("Resolve(%q): unexpected error %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("Resolve(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
