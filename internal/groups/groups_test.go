package groups

import (
	"strings"
	"testing"
)

func TestNewInviteCode(t *testing.T) {
	const validChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567" // base32 (RFC 4648), uppercased

	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		code := newInviteCode()
		if len(code) != 6 {
			t.Fatalf("code %q has length %d, want 6", code, len(code))
		}
		if code != strings.ToUpper(code) {
			t.Fatalf("code %q is not uppercase", code)
		}
		for _, ch := range code {
			if !strings.ContainsRune(validChars, ch) {
				t.Fatalf("code %q has invalid char %q", code, ch)
			}
		}
		seen[code] = true
	}
	// Not a strict guarantee, but 200 draws from 32^6 should not collide down to
	// a handful — a tiny unique set would signal a broken RNG.
	if len(seen) < 190 {
		t.Errorf("only %d unique codes from 200 draws — suspicious", len(seen))
	}
}
