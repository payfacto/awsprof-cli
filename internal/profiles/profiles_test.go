package profiles

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func find(ps []Profile, name string) *Profile {
	for i := range ps {
		if ps[i].Name == name {
			return &ps[i]
		}
	}
	return nil
}

func TestListFrom_ClassifiesAndParses(t *testing.T) {
	dir := t.TempDir()
	cfg := writeFile(t, dir, "config", `
[default]
region = us-east-1

[sso-session payfacto]
sso_start_url = https://payfacto.awsapps.com/start
sso_region = us-east-1

[profile payfacto-synapse-admin]
sso_session = payfacto
sso_account_id = 111122223333
sso_role_name = Admin
region = us-east-1

[profile legacy-sso]
sso_start_url = https://old.awsapps.com/start
sso_region = us-west-2
sso_account_id = 444455556666
sso_role_name = ReadOnly

[profile role-prof]
role_arn = arn:aws:iam::123:role/thing
source_profile = default

[profile proc-prof]
credential_process = /usr/bin/cred-helper
`)
	cred := writeFile(t, dir, "credentials", `
[static-prof]
aws_access_key_id = AKIA...
aws_secret_access_key = secret
`)

	ps, err := ListFrom(cfg, cred)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sso := find(ps, "payfacto-synapse-admin")
	if sso == nil || sso.Type != TypeSSO || sso.SSO == nil {
		t.Fatalf("payfacto-synapse-admin not classified as SSO: %+v", sso)
	}
	if sso.SSO.StartURL != "https://payfacto.awsapps.com/start" || sso.SSO.Region != "us-east-1" {
		t.Fatalf("sso-session not resolved: %+v", sso.SSO)
	}
	if sso.SSO.AccountID != "111122223333" || sso.SSO.RoleName != "Admin" {
		t.Fatalf("sso account/role wrong: %+v", sso.SSO)
	}

	legacy := find(ps, "legacy-sso")
	if legacy == nil || legacy.Type != TypeSSO || legacy.SSO.StartURL != "https://old.awsapps.com/start" {
		t.Fatalf("legacy sso wrong: %+v", legacy)
	}

	if p := find(ps, "role-prof"); p == nil || p.Type != TypeAssumeRole {
		t.Fatalf("role-prof wrong: %+v", p)
	}
	if p := find(ps, "proc-prof"); p == nil || p.Type != TypeProcess {
		t.Fatalf("proc-prof wrong: %+v", p)
	}
	if p := find(ps, "static-prof"); p == nil || p.Type != TypeStatic {
		t.Fatalf("static-prof wrong: %+v", p)
	}
	if p := find(ps, "default"); p == nil {
		t.Fatalf("default profile missing")
	}
}

func TestListFrom_MissingFilesAreTolerated(t *testing.T) {
	ps, err := ListFrom(filepath.Join(t.TempDir(), "none"), filepath.Join(t.TempDir(), "none"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ps) != 0 {
		t.Fatalf("expected no profiles, got %v", ps)
	}
}

func TestListFrom_SortedByName(t *testing.T) {
	dir := t.TempDir()
	cfg := writeFile(t, dir, "config", "[profile zeta]\n[profile alpha]\n")
	ps, err := ListFrom(cfg, filepath.Join(dir, "none"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 || ps[0].Name != "alpha" || ps[1].Name != "zeta" {
		t.Fatalf("not sorted: %v", ps)
	}
}
