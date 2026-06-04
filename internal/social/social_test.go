package social

import (
	"testing"
	"time"
)

func TestToString(t *testing.T) {
	ts := time.Date(2026, 6, 4, 9, 30, 0, 0, time.UTC)
	if got := toString(ts); got != "2026-06-04T09:30:00Z" {
		t.Errorf("toString(time) = %q", got)
	}
	if got := toString("x"); got != "x" {
		t.Errorf("toString(string) = %q", got)
	}
	if got := toString(nil); got != "" {
		t.Errorf("toString(nil) = %q, want empty", got)
	}
}
