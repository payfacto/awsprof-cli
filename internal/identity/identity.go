// Package identity resolves and verifies the caller identity for a profile.
package identity

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
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
// meaning the caller should run an SSO login rather than fix permissions.
//
// The classification is two-stage:
//
//  1. If err is a smithy API error, the request reached AWS and got a
//     response, so credentials resolved fine. That makes it an
//     authorization/service problem (for example AccessDenied, possibly with
//     an AWSReservedSSO_ assumed-role ARN in the message) - never a login
//     problem - so we return false.
//  2. Otherwise err is a credential-resolution failure. We match it against
//     SSO-token-expiry wording; a bare mention of "sso" is not enough.
//
// The exact text the SSO credential provider returns for an expired token is
// still a deferred live-confirmation item; the wording matched below is the
// starting point.
func NeedsLogin(err error) bool {
	if err == nil {
		return false
	}
	// A response came back from AWS, so credentials resolved: not a login issue.
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "expired") ||
		strings.Contains(msg, "run aws sso login") ||
		strings.Contains(msg, "refresh") ||
		strings.Contains(msg, "invalid_grant") ||
		strings.Contains(msg, "sso session") ||
		strings.Contains(msg, "sso token")
}
