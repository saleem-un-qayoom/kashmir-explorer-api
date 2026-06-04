package review

import (
	"testing"
	"time"
)

func TestTargetTable(t *testing.T) {
	if got := targetTable("trek"); got != "treks" {
		t.Errorf("targetTable(trek) = %q, want treks", got)
	}
	if got := targetTable("destination"); got != "destinations" {
		t.Errorf("targetTable(destination) = %q, want destinations", got)
	}
	// Defaults to destinations for anything unexpected (never user-controlled).
	if got := targetTable(""); got != "destinations" {
		t.Errorf("targetTable(\"\") = %q, want destinations", got)
	}
}

func TestToString(t *testing.T) {
	ts := time.Date(2026, 6, 4, 9, 30, 0, 0, time.UTC)
	if got := toString(ts); got != "2026-06-04T09:30:00Z" {
		t.Errorf("toString(time) = %q", got)
	}
	if got := toString("already-a-string"); got != "already-a-string" {
		t.Errorf("toString(string) = %q", got)
	}
	if got := toString(nil); got != "" {
		t.Errorf("toString(nil) = %q, want empty", got)
	}
	if got := toString(42); got != "" {
		t.Errorf("toString(int) = %q, want empty", got)
	}
}
