package jwt

import "testing"

func newTestIssuer() *Issuer {
	return NewIssuer("access-secret", "refresh-secret", 24, 30)
}

func TestIssueVerifyRoundTrip(t *testing.T) {
	iss := newTestIssuer()
	access, refresh, err := iss.Issue("user-123", "admin")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatal("expected non-empty tokens")
	}

	claims, err := iss.Verify(access)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want user-123", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, want admin", claims.Role)
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	access, _, err := newTestIssuer().Issue("u", "user")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	other := NewIssuer("different-secret", "refresh-secret", 24, 30)
	if _, err := other.Verify(access); err == nil {
		t.Fatal("expected verification to fail under a different secret")
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	// Negative TTL → token already expired the moment it is issued.
	iss := NewIssuer("access-secret", "refresh-secret", -1, 30)
	access, _, err := iss.Issue("u", "user")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if _, err := iss.Verify(access); err == nil {
		t.Fatal("expected expired token to fail verification")
	}
}

func TestRefreshTokenIsNotAValidAccessToken(t *testing.T) {
	// Verify only trusts the access secret; a refresh token (signed with the
	// refresh secret) must not pass as an access token.
	iss := newTestIssuer()
	_, refresh, err := iss.Issue("u", "user")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if _, err := iss.Verify(refresh); err == nil {
		t.Fatal("expected refresh token to be rejected by Verify")
	}
}

func TestVerifyRejectsTampered(t *testing.T) {
	access, _, err := newTestIssuer().Issue("u", "user")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	tampered := access[:len(access)-2] + "xy"
	if _, err := newTestIssuer().Verify(tampered); err == nil {
		t.Fatal("expected tampered token to fail verification")
	}
}
