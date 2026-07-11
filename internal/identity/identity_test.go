package identity

import (
	"errors"
	"testing"

	"github.com/aws/smithy-go"
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

// A real SSO-assumed-role AccessDenied is a smithy API error: the request
// reached AWS, so credentials resolved. Even though its message contains both
// "sso" (in the AWSReservedSSO_ ARN) and "refresh", it must not be treated as
// needs-login - telling the user to run aws sso login would not fix an SCP or
// permissions deny.
func TestNeedsLogin_APIErrorNotLogin(t *testing.T) {
	denied := &smithy.GenericAPIError{
		Code:    "AccessDenied",
		Message: "User arn:aws:sts::111122223333:assumed-role/AWSReservedSSO_Admin_abc/alice is not authorized; refresh and retry",
	}
	if NeedsLogin(denied) {
		t.Error("API AccessDenied (SSO-assumed-role) must not be treated as needs-login")
	}
}
