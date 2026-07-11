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

// NeedsLogin reports whether err indicates a missing or expired SSO token
// (as opposed to an authorization failure or a non-SSO credential problem).
func NeedsLogin(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "sso") {
		return false
	}
	return strings.Contains(msg, "expired") ||
		strings.Contains(msg, "run aws sso login") ||
		strings.Contains(msg, "refresh") ||
		strings.Contains(msg, "invalid_grant")
}
