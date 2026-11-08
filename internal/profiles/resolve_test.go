package profiles

import (
	"reflect"
	"testing"
)

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
		got, err := Resolve(c.in, Resolver{Prefixes: c.prefixes}, names)
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
	if got, _ := Resolve("data", Resolver{Prefixes: []string{"corp-", "payfacto-"}}, names); got != "corp-data" {
		t.Errorf(`prefixes [corp-,payfacto-]: got %q, want "corp-data"`, got)
	}
	if got, _ := Resolve("data", Resolver{Prefixes: []string{"payfacto-", "corp-"}}, names); got != "payfacto-data" {
		t.Errorf(`prefixes [payfacto-,corp-]: got %q, want "payfacto-data"`, got)
	}
}

func TestResolve_Aliases(t *testing.T) {
	names := []string{"payfacto-synapse-admin", "payfacto-sandbox-admin", "sa"}
	r := Resolver{
		Prefixes: []string{"payfacto-"},
		Aliases: map[string]string{
			"sa":     "payfacto-synapse-admin", // shadowed by the exact profile "sa"
			"syn":    "payfacto-synapse-admin", // full target name
			"ba":     "sandbox-admin",          // short target, resolved via prefix
			"broken": "payfacto-missing",
		},
	}
	cases := []struct {
		in, want, wantErrMsg string
	}{
		{"sa", "sa", ""},
		{"syn", "payfacto-synapse-admin", ""},
		{"ba", "payfacto-sandbox-admin", ""},
		{"broken", "", `alias "broken" points to unknown profile "payfacto-missing"`},
	}
	for _, c := range cases {
		got, err := Resolve(c.in, r, names)
		if c.wantErrMsg != "" {
			if err == nil || err.Error() != c.wantErrMsg {
				t.Errorf("Resolve(%q): error = %v, want %q", c.in, err, c.wantErrMsg)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("Resolve(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

func TestResolve_AliasBeatsPrefix(t *testing.T) {
	names := []string{"payfacto-sp", "payfacto-synapse-poweruser"}
	r := Resolver{Prefixes: []string{"payfacto-"}, Aliases: map[string]string{"sp": "synapse-poweruser"}}
	if got, _ := Resolve("sp", r, names); got != "payfacto-synapse-poweruser" {
		t.Errorf("alias must win over prefix match, got %q", got)
	}
}

func TestAliasesByProfile(t *testing.T) {
	names := []string{"payfacto-synapse-admin", "payfacto-sandbox-admin"}
	r := Resolver{
		Prefixes: []string{"payfacto-"},
		Aliases: map[string]string{
			"sa":     "synapse-admin",
			"syn":    "payfacto-synapse-admin",
			"ba":     "sandbox-admin",
			"broken": "nope",
		},
	}
	got, broken := AliasesByProfile(r, names)
	want := map[string][]string{
		"payfacto-synapse-admin": {"sa", "syn"},
		"payfacto-sandbox-admin": {"ba"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("index = %v, want %v", got, want)
	}
	if len(broken) != 1 || broken[0].Error() != `alias "broken" points to unknown profile "nope"` {
		t.Errorf("broken = %v", broken)
	}
}

func TestResolve_AliasesDoNotChain(t *testing.T) {
	names := []string{"payfacto-synapse-admin"}
	r := Resolver{Prefixes: []string{"payfacto-"}, Aliases: map[string]string{"a": "b", "b": "synapse-admin"}}
	if _, err := Resolve("a", r, names); err == nil {
		t.Fatal("alias pointing at another alias must not resolve")
	}
}
