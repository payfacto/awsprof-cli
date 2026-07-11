// Package identity resolves and verifies the caller identity for a profile.
package identity

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Identity is the resolved AWS caller identity.
type Identity struct {
	Account string
	Arn     string
}

// Check loads shared config for profile and calls STS GetCallerIdentity.
func Check(ctx context.Context, profile string) (Identity, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
	if err != nil {
		return Identity{}, err
	}
	out, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return Identity{}, err
	}
	return Identity{Account: aws.ToString(out.Account), Arn: aws.ToString(out.Arn)}, nil
}

// NeedsLogin reports whether err indicates a missing or expired SSO token,
// meaning the caller should run an SSO login rather than surface the error.
//
// It matches positive SSO-token-failure signals in the error text. Those appear
// in credential-resolution failures - including when a stale token's refresh
// fails, where the STS error wraps an ssooidc InvalidGrantException (itself a
// smithy API error). They do NOT appear in a plain STS authorization denial
// such as AccessDenied - even one whose message embeds an AWSReservedSSO_
// assumed-role ARN - so a permissions problem never triggers a spurious login.
//
// The signals were confirmed against a real expired-SSO error observed in a
// live smoke test. Note: an earlier version short-circuited to false on any
// smithy.APIError in the chain, which wrongly classified the common
// expired-token case (its chain contains an ssooidc InvalidGrantException) as
// "not a login problem". Match on the token-failure wording instead.
func NeedsLogin(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	signals := []string{
		"refresh cached sso token",
		"refresh sso token",
		"sso token",
		"sso session has expired",
		"invalidgrant", // ssooidc InvalidGrantException (stale token refresh)
		"expiredtoken", // ssooidc ExpiredTokenException
		"token has expired",
		"run aws sso login",
		"no cached sso",
	}
	for _, s := range signals {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}
