package sso

import (
	"context"
	"errors"
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
}

func (f *fakeOIDC) RegisterClient(_ context.Context, _ *ssooidc.RegisterClientInput, _ ...func(*ssooidc.Options)) (*ssooidc.RegisterClientOutput, error) {
	return &ssooidc.RegisterClientOutput{ClientId: aws.String("cid"), ClientSecret: aws.String("csec")}, nil
}

func (f *fakeOIDC) StartDeviceAuthorization(_ context.Context, _ *ssooidc.StartDeviceAuthorizationInput, _ ...func(*ssooidc.Options)) (*ssooidc.StartDeviceAuthorizationOutput, error) {
	return &ssooidc.StartDeviceAuthorizationOutput{
		DeviceCode:              aws.String("dev"),
		UserCode:                aws.String("USER-CODE"),
		VerificationUriComplete: aws.String("https://device.sso/verify?code=USER-CODE"),
		Interval:                1,
		ExpiresIn:               600,
	}, nil
}

func (f *fakeOIDC) CreateToken(_ context.Context, _ *ssooidc.CreateTokenInput, _ ...func(*ssooidc.Options)) (*ssooidc.CreateTokenOutput, error) {
	f.createCalls++
	if f.denied {
		return nil, &ssotypes.AccessDeniedException{}
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
