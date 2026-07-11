package identity

import (
	"errors"
	"testing"
)

func TestNeedsLogin(t *testing.T) {
	yes := []error{
		errors.New("failed to refresh cached SSO token, the SSO session has expired"),
		errors.New("the SSO session associated with this profile has expired or is otherwise invalid; to refresh this SSO session run aws sso login"),
	}
	for _, e := range yes {
		if !NeedsLogin(e) {
			t.Errorf("NeedsLogin should be true for: %v", e)
		}
	}
	no := []error{
		nil,
		errors.New("AccessDenied: not authorized to perform sts:GetCallerIdentity"),
		errors.New("no EC2 IMDS role found"),
	}
	for _, e := range no {
		if NeedsLogin(e) {
			t.Errorf("NeedsLogin should be false for: %v", e)
		}
	}
}
