package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/charmbracelet/lipgloss"
	"github.com/payfacto/awsprof-cli/internal/config"
	"github.com/payfacto/awsprof-cli/internal/envcolor"
	"github.com/payfacto/awsprof-cli/internal/identity"
	"github.com/payfacto/awsprof-cli/internal/profiles"
	"github.com/payfacto/awsprof-cli/internal/shell"
	"github.com/payfacto/awsprof-cli/internal/sso"
	"github.com/pkg/browser"
)

// resolveTarget maps a typed name to an existing profile using configured
// prefixes. On an unknown name it prints the available profiles to stderr
// before returning the error, so callers never need to duplicate that step.
func resolveTarget(name string) (profiles.Profile, error) {
	ps, err := profiles.List()
	if err != nil {
		return profiles.Profile{}, err
	}
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return profiles.Profile{}, err
	}
	names := make([]string, len(ps))
	for i, p := range ps {
		names[i] = p.Name
	}
	resolved, err := profiles.Resolve(name, cfg.Prefixes, names)
	if err != nil {
		printProfiles(ps)
		return profiles.Profile{}, err
	}
	for _, p := range ps {
		if p.Name == resolved {
			return p, nil
		}
	}
	return profiles.Profile{}, fmt.Errorf("unknown profile %q", name)
}

// resolveTargetForTest is a thin seam used by tests to exercise resolution
// without pulling in the rest of the activation flow (identity check, SSO).
func resolveTargetForTest(name string) error {
	_, err := resolveTarget(name)
	return err
}

func printProfiles(ps []profiles.Profile) {
	r := lipgloss.NewRenderer(os.Stderr)
	fmt.Fprintln(os.Stderr, "Available profiles:")
	for _, p := range ps {
		fmt.Fprintf(os.Stderr, "  %s\n", envcolor.Render(p.Name, r))
	}
}

// activate resolves name to a profile, logs in via SSO if needed, verifies
// the resulting identity, and on success prints the shell export line to
// stdout (the only thing ever written there) and the identity to stderr.
// On any failure it returns an error and prints no export.
func activate(ctx context.Context, name string, sh shell.Shell) error {
	prof, err := resolveTarget(name)
	if err != nil {
		return err
	}

	id, err := identity.Check(ctx, prof.Name)
	if err != nil {
		if prof.Type == profiles.TypeSSO && prof.SSO != nil && identity.NeedsLogin(err) {
			if loginErr := ssoLogin(ctx, *prof.SSO); loginErr != nil {
				return loginErr
			}
			id, err = identity.Check(ctx, prof.Name)
		}
		if err != nil {
			return err
		}
	}

	// stdout: only the export line (for the shell wrapper to eval).
	fmt.Fprintln(os.Stdout, sh.ExportLine(prof.Name))
	// stderr: human confirmation.
	fmt.Fprintf(os.Stderr, "AWS_PROFILE=%s\nAccount %s  %s\n", prof.Name, id.Account, id.Arn)
	return nil
}

// ssoLogin runs the device-authorization flow for an SSO profile and caches
// the resulting token where the AWS SDK/CLI expect to find it.
func ssoLogin(ctx context.Context, sc profiles.SSOConfig) error {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(sc.Region))
	if err != nil {
		return err
	}
	client := ssooidc.NewFromConfig(cfg)
	tok, err := sso.Login(ctx, client, browser.OpenURL, sc, time.Now)
	if err != nil {
		return err
	}
	path, err := sso.CacheFilePath(sc.SessionName, sc.StartURL)
	if err != nil {
		return err
	}
	return sso.WriteToken(path, tok)
}
