package tests

import (
	"testing"

	"webtyp.com/fmt"
)

func TestPresence(t *testing.T) {
	partsList := []fmt.KeyValue{
		{Key: "u1", Value: "Alice"},
		{Key: "u2", Value: "Bob"},
	}
	env, err := setupTestEnv("t1", 30, partsList)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// u2 sends Heartbeat at t=100s (100,000,000,000 ns)
	env.clockTime = 100_000_000_000
	_, err = env.module.Heartbeat("t1", "u2")
	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	// u1 checks ListParticipants at t=189s (diff = 89s < 90s => online)
	env.clockTime = 189_000_000_000
	parts, err := env.module.ListParticipants("t1", "u1")
	if err != nil {
		t.Fatalf("ListParticipants failed: %v", err)
	}
	if len(parts) != 1 || !parts[0].Online {
		t.Fatalf("expected u2 online at 89s diff, got %v", parts)
	}

	// u1 checks ListParticipants at t=191s (diff = 91s >= 90s => offline)
	env.clockTime = 191_000_000_000
	partsOffline, err := env.module.ListParticipants("t1", "u1")
	if err != nil {
		t.Fatalf("ListParticipants failed: %v", err)
	}
	if len(partsOffline) != 1 || partsOffline[0].Online {
		t.Fatalf("expected u2 offline at 91s diff, got %v", partsOffline)
	}
}
