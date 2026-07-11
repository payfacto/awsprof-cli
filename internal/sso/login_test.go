package sso

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
	"github.com/payfacto/awsprof-cli/internal/profiles"
)

type fakeOIDC struct {
	createCalls int
	failAfter   int
	denied      bool
	expired     bool
	genericErr  bool
	expiresIn   int32 // StartDeviceAuthorization ExpiresIn; 0 => default 600
}

func (f *fakeOIDC) RegisterClient(_ context.Context, _ *ssooidc.RegisterClientInput, _ ...func(*ssooidc.Options)) (*ssooidc.RegisterClientOutput, error) {
	return &ssooidc.RegisterClientOutput{ClientId: aws.String("cid"), ClientSecret: aws.String("csec")}, nil
}

func (f *fakeOIDC) StartDeviceAuthorization(_ context.Context, _ *ssooidc.StartDeviceAuthorizationInput, _ ...func(*ssooidc.Options)) (*ssooidc.StartDeviceAuthorizationOutput, error) {
	exp := int32(600)
	if f.expiresIn != 0 {
		exp = f.expiresIn
	}
	return &ssooidc.StartDeviceAuthorizationOutput{
		DeviceCode:              aws.String("dev"),
		UserCode:                aws.String("USER-CODE"),
		VerificationUriComplete: aws.String("https://device.sso/verify?code=USER-CODE"),
		Interval:                1,
		ExpiresIn:               exp,
	}, nil
}

func (f *fakeOIDC) CreateToken(_ context.Context, _ *ssooidc.CreateTokenInput, _ ...func(*ssooidc.Options)) (*ssooidc.CreateTokenOutput, error) {
	f.createCalls++
	if f.denied {
		return nil, &ssotypes.AccessDeniedException{}
	}
	if f.expired {
		return nil, &ssotypes.ExpiredTokenException{}
	}
	if f.genericErr {
		return nil, errors.New("boom")
	}
	if f.createCalls <= f.failAfter {
		return nil, &ssotypes.AuthorizationPendingException{}
	}
	return &ssooidc.CreateTokenOutput{AccessToken: aws.String("access-tok"), ExpiresIn: 3600}, nil
}

func TestLogin_SucceedsAfterPending(t *testing.T) {
	var opened string
	cfg := profiles.SSOConfig{SessionName: "payfacto", StartURL: "https://x/start", Region: "us-east-1"}
	tok, err := Login(context.Background(), &fakeOIDC{failAfter: 2}, func(u string) error { opened = u; return nil }, cfg, time.Now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "access-tok" {
		t.Fatalf("token = %+v", tok)
	}
	if opened == "" {
		t.Fatalf("browser opener not called")
	}
	if tok.StartURL != "https://x/start" || tok.Region != "us-east-1" {
		t.Fatalf("token missing session info: %+v", tok)
	}
}

func TestLogin_AccessDenied(t *testing.T) {
	cfg := profiles.SSOConfig{StartURL: "https://x/start", Region: "us-east-1"}
	_, err := Login(context.Background(), &fakeOIDC{denied: true}, func(string) error { return nil }, cfg, time.Now)
	if err == nil || !errors.Is(err, ErrLoginDenied) {
		t.Fatalf("expected ErrLoginDenied, got %v", err)
	}
}

// TestLogin_ExpiredToken: CreateToken returns ExpiredTokenException on the
// first call, so Login maps it to ErrLoginTimeout and returns without waiting.
func TestLogin_ExpiredToken(t *testing.T) {
	cfg := profiles.SSOConfig{StartURL: "https://x/start", Region: "us-east-1"}
	_, err := Login(context.Background(), &fakeOIDC{expired: true}, func(string) error { return nil }, cfg, time.Now)
	if err == nil || !errors.Is(err, ErrLoginTimeout) {
		t.Fatalf("expected ErrLoginTimeout, got %v", err)
	}
}

// TestLogin_GenericError: a non-classified CreateToken error is wrapped and
// returned immediately, and is neither ErrLoginDenied nor ErrLoginTimeout.
func TestLogin_GenericError(t *testing.T) {
	cfg := profiles.SSOConfig{StartURL: "https://x/start", Region: "us-east-1"}
	_, err := Login(context.Background(), &fakeOIDC{genericErr: true}, func(string) error { return nil }, cfg, time.Now)
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if errors.Is(err, ErrLoginDenied) || errors.Is(err, ErrLoginTimeout) {
		t.Fatalf("expected a generic wrapped error, got sentinel: %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped cause %q, got %v", "boom", err)
	}
}

// TestLogin_DeadlineTimeout: with ExpiresIn=1, a now() that jumps 10s after the
// deadline is computed makes the loop's deadline check fire on the first
// iteration, before any CreateToken poll or time.After wait. Instant.
func TestLogin_DeadlineTimeout(t *testing.T) {
	cfg := profiles.SSOConfig{StartURL: "https://x/start", Region: "us-east-1"}
	base := time.Now()
	calls := 0
	now := func() time.Time {
		calls++
		if calls == 1 {
			return base // used to compute deadline = base + 1s
		}
		return base.Add(10 * time.Second) // loop check: already past deadline
	}
	f := &fakeOIDC{failAfter: 1_000_000, expiresIn: 1}
	_, err := Login(context.Background(), f, func(string) error { return nil }, cfg, now)
	if err == nil || !errors.Is(err, ErrLoginTimeout) {
		t.Fatalf("expected ErrLoginTimeout, got %v", err)
	}
	if f.createCalls != 0 {
		t.Fatalf("expected no CreateToken calls before deadline, got %d", f.createCalls)
	}
}

// TestLogin_ContextCanceled: CreateToken always returns pending; the context is
// already canceled, so after the first poll the select observes ctx.Done()
// (ready) ahead of the 1s timer and returns ctx.Err(). Instant.
func TestLogin_ContextCanceled(t *testing.T) {
	cfg := profiles.SSOConfig{StartURL: "https://x/start", Region: "us-east-1"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &fakeOIDC{failAfter: 1_000_000} // always pending
	_, err := Login(ctx, f, func(string) error { return nil }, cfg, time.Now)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
