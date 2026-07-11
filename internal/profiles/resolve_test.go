package profiles

import "testing"

func TestResolve(t *testing.T) {
	names := []string{"payfacto-synapse-admin", "corp-data", "default"}
	cases := []struct {
		in         string
		prefixes   []string
		want       string
		wantErr    bool
		wantErrMsg string
	}{
		{"payfacto-synapse-admin", []string{"payfacto-"}, "payfacto-synapse-admin", false, ""},
		{"synapse-admin", []string{"payfacto-"}, "payfacto-synapse-admin", false, ""},
		{"data", []string{"payfacto-", "corp-"}, "corp-data", false, ""},
		{"nope", []string{"payfacto-"}, "", true, `unknown profile "nope"`},
		{"default", nil, "default", false, ""},
	}
	for _, c := range cases {
		got, err := Resolve(c.in, c.prefixes, names)
		if c.wantErr {
			if err == nil {
				t.Errorf("Resolve(%q): expected error", c.in)
				continue
			}
			if err.Error() != c.wantErrMsg {
				t.Errorf("Resolve(%q): error = %q, want %q", c.in, err.Error(), c.wantErrMsg)
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

func TestResolve_PrefixOrderWins(t *testing.T) {
	names := []string{"payfacto-data", "corp-data"}
	if got, _ := Resolve("data", []string{"corp-", "payfacto-"}, names); got != "corp-data" {
		t.Errorf(`prefixes [corp-,payfacto-]: got %q, want "corp-data"`, got)
	}
	if got, _ := Resolve("data", []string{"payfacto-", "corp-"}, names); got != "payfacto-data" {
		t.Errorf(`prefixes [payfacto-,corp-]: got %q, want "payfacto-data"`, got)
	}
}
