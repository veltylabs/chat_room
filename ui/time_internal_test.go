package ui

import (
	"testing"
	"webtyp.com/time"
)

func TestClockAndShortTime(t *testing.T) {
	zone, _ := time.LoadZone("America/Santiago")
	s := &source{zone: zone}

	if got := s.clock(0); got != "" {
		t.Fatalf("expected empty string for clock(0), got %q", got)
	}

	if got := s.shortTime(0); got != "" {
		t.Fatalf("expected empty string for shortTime(0), got %q", got)
	}

	fixedNs := time.Nano(1700000000000000000)
	c := s.clock(fixedNs)
	if len(c) != 5 {
		t.Fatalf("expected clock format 'HH:MM' (length 5), got %q", c)
	}

	st := s.shortTime(fixedNs)
	if st == "" {
		t.Fatalf("expected non-empty shortTime for timestamp, got empty")
	}

	// New test: shortTime of 2019-07-16 01:30 UTC with Santiago on a day that is not today -> "2019-07-15".
	tsUTC := time.Unix(1563240600)
	got := s.shortTime(time.Nano(tsUTC) * 1000000000)
	if got != "2019-07-15" {
		t.Fatalf("expected shortTime for 2019-07-16 01:30 UTC to be '2019-07-15' in Santiago, got %q", got)
	}
}
