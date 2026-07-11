package sso

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
	"github.com/payfacto/awsprof-cli/internal/profiles"
)

// ErrLoginDenied is returned when the user denies the device authorization.
var ErrLoginDenied = errors.New("sso login denied")

// ErrLoginTimeout is returned when the device code expires before authorization.
var ErrLoginTimeout = errors.New("sso login timed out")

// OIDCClient is the subset of the ssooidc client the login flow needs.
type OIDCClient interface {
	RegisterClient(context.Context, *ssooidc.RegisterClientInput, ...func(*ssooidc.Options)) (*ssooidc.RegisterClientOutput, error)
	StartDeviceAuthorization(context.Context, *ssooidc.StartDeviceAuthorizationInput, ...func(*ssooidc.Options)) (*ssooidc.StartDeviceAuthorizationOutput, error)
	CreateToken(context.Context, *ssooidc.CreateTokenInput, ...func(*ssooidc.Options)) (*ssooidc.CreateTokenOutput, error)
}

// Compile-time assertion that the real ssooidc client satisfies OIDCClient.
var _ OIDCClient = (*ssooidc.Client)(nil)

// Opener opens a URL in the user's browser.
type Opener func(url string) error

// Login runs the device-authorization grant and returns an access token.
func Login(ctx context.Context, c OIDCClient, open Opener, cfg profiles.SSOConfig, now func() time.Time) (Token, error) {
	reg, err := c.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String("awsprof"),
		ClientType: aws.String("public"),
		Scopes:     []string{"sso:account:access"},
	})
	if err != nil {
		return Token{}, fmt.Errorf("register client: %w", err)
	}

	auth, err := c.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     reg.ClientId,
		ClientSecret: reg.ClientSecret,
		StartUrl:     aws.String(cfg.StartURL),
	})
	if err != nil {
		return Token{}, fmt.Errorf("start device authorization: %w", err)
	}

	url := aws.ToString(auth.VerificationUriComplete)
	fmt.Fprintf(os.Stderr, "Opening browser to authorize SSO login...\nIf it does not open, visit: %s\nUser code: %s\n", url, aws.ToString(auth.UserCode))
	if err := open(url); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not open browser: %v\n", err)
	}

	interval := time.Duration(auth.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := now().Add(time.Duration(auth.ExpiresIn) * time.Second)

	for {
		if now().After(deadline) {
			return Token{}, ErrLoginTimeout
		}
		out, err := c.CreateToken(ctx, &ssooidc.CreateTokenInput{
			ClientId:     reg.ClientId,
			ClientSecret: reg.ClientSecret,
			DeviceCode:   auth.DeviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		})
		if err == nil {
			return Token{
				AccessToken:  aws.ToString(out.AccessToken),
				ExpiresAt:    now().Add(time.Duration(out.ExpiresIn) * time.Second),
				StartURL:     cfg.StartURL,
				Region:       cfg.Region,
				ClientID:     aws.ToString(reg.ClientId),
				ClientSecret: aws.ToString(reg.ClientSecret),
				RefreshToken: aws.ToString(out.RefreshToken),
			}, nil
		}

		var pending *ssotypes.AuthorizationPendingException
		var slow *ssotypes.SlowDownException
		var denied *ssotypes.AccessDeniedException
		var expired *ssotypes.ExpiredTokenException
		switch {
		case errors.As(err, &pending):
			// keep polling
		case errors.As(err, &slow):
			interval += 5 * time.Second
		case errors.As(err, &denied):
			return Token{}, ErrLoginDenied
		case errors.As(err, &expired):
			return Token{}, ErrLoginTimeout
		default:
			return Token{}, fmt.Errorf("create token: %w", err)
		}

		select {
		case <-ctx.Done():
			return Token{}, ctx.Err()
		case <-time.After(interval):
		}
	}
}
