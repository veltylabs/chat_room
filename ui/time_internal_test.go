package ui

import (
	"testing"
)

func TestClockAndShortTime(t *testing.T) {
	if got := clock(0); got != "" {
		t.Fatalf("expected empty string for clock(0), got %q", got)
	}

	if got := shortTime(0); got != "" {
		t.Fatalf("expected empty string for shortTime(0), got %q", got)
	}

	fixedNs := int64(1700000000000000000)
	c := clock(fixedNs)
	if len(c) != 5 {
		t.Fatalf("expected clock format 'HH:MM' (length 5), got %q", c)
	}

	st := shortTime(fixedNs)
	if st == "" {
		t.Fatalf("expected non-empty shortTime for timestamp, got empty")
	}
}
