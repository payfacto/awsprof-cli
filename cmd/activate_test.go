package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/payfacto/awsprof-cli/internal/profiles"
)

// activate() must fail fast (before any network) on an unknown profile and
// must not print an export to stdout.
func TestActivate_UnknownProfileNoExport(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config")
	if err := os.WriteFile(cfgPath, []byte("[profile payfacto-real]\nsso_start_url=https://x/start\nsso_region=us-east-1\nsso_account_id=1\nsso_role_name=r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_CONFIG_FILE", cfgPath)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "creds"))

	out := captureStdout(t, func() {
		if _, err := resolveTarget("does-not-exist"); err == nil {
			t.Fatal("expected error for unknown profile")
		}
	})
	if out != "" {
		t.Fatalf("expected no stdout on failure, got %q", out)
	}
}

func TestRegionHint(t *testing.T) {
	base := errors.New("boom")

	ssoNoRegion := profiles.Profile{Name: "p", Type: profiles.TypeSSO, SSO: &profiles.SSOConfig{Region: ""}}
	got := regionHint(ssoNoRegion, base)
	if !strings.Contains(got.Error(), "no region") || !errors.Is(got, base) {
		t.Errorf("SSO-without-region should be annotated and wrap the cause; got %v", got)
	}

	ssoWithRegion := profiles.Profile{Name: "p", Type: profiles.TypeSSO, SSO: &profiles.SSOConfig{Region: "us-east-1"}}
	if got := regionHint(ssoWithRegion, base); got != base {
		t.Errorf("SSO-with-region should pass the error through unchanged; got %v", got)
	}

	static := profiles.Profile{Name: "p", Type: profiles.TypeStatic}
	if got := regionHint(static, base); got != base {
		t.Errorf("non-SSO should pass the error through unchanged; got %v", got)
	}

	if regionHint(ssoNoRegion, nil) != nil {
		t.Errorf("nil error should stay nil")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = orig }()
	fn()
	_ = w.Close()
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	if n < 0 {
		n = 0
	}
	return string(buf[:n])
}
