package tests

import (
	"testing"

	"webtyp.com/fmt"
)

func TestPurgeExpired(t *testing.T) {
	partsList := []fmt.KeyValue{
		{Key: "u1", Value: "Alice"},
		{Key: "u2", Value: "Bob"},
	}
	// Retention = 2 days
	env, err := setupTestEnv("t1", 2, partsList)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	room, err := env.module.OpenDirect("t1", "u1", "u2")
	if err != nil {
		t.Fatalf("OpenDirect failed: %v", err)
	}

	// Message 1 at t=0
	env.clockTime = 1_000_000_000_000
	_, err = env.module.SendMessage("t1", "u1", room.RoomId, "Old message")
	if err != nil {
		t.Fatalf("SendMessage old failed: %v", err)
	}

	// Message 2 at t = 3 days later
	threeDaysNanos := int64(3) * 24 * 3600 * 1_000_000_000
	env.clockTime += threeDaysNanos
	_, err = env.module.SendMessage("t1", "u1", room.RoomId, "New message")
	if err != nil {
		t.Fatalf("SendMessage new failed: %v", err)
	}

	// PurgeExpired should remove 1 expired message
	purged, err := env.module.PurgeExpired("t1")
	if err != nil {
		t.Fatalf("PurgeExpired failed: %v", err)
	}
	if purged != 1 {
		t.Fatalf("expected 1 message purged, got %d", purged)
	}

	msgs, err := env.module.ListMessages("t1", "u1", room.RoomId, "", 0)
	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Body != "New message" {
		t.Fatalf("expected only 'New message' remaining, got %v", msgs)
	}
}
