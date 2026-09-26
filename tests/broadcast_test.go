package tests

import (
	"testing"

	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/fmt"
)

func TestBroadcastRoom(t *testing.T) {
	partsList := []fmt.KeyValue{
		{Key: "u1", Value: "Alice"},
		{Key: "u2", Value: "Bob"},
	}
	env, err := setupTestEnv("t1", 30, partsList)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// ListRooms automatically ensures broadcast room creation
	rooms1, err := env.module.ListRooms("t1", "u1")
	if err != nil {
		t.Fatalf("ListRooms failed: %v", err)
	}

	if len(rooms1) != 1 || rooms1[0].Kind != chatroom.KindBroadcast {
		t.Fatalf("expected 1 broadcast room, got %v", rooms1)
	}
	broadcastID := rooms1[0].RoomId

	// Calling ListRooms for u2 returns the same broadcast room
	rooms2, err := env.module.ListRooms("t1", "u2")
	if err != nil {
		t.Fatalf("ListRooms failed: %v", err)
	}
	if len(rooms2) != 1 || rooms2[0].RoomId != broadcastID {
		t.Fatalf("expected same broadcast room ID, got %s vs %s", broadcastID, rooms2[0].RoomId)
	}

	// All participants can send and read messages
	_, err = env.module.SendMessage("t1", "u1", broadcastID, "Hello broadcast")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	msgs, err := env.module.ListMessages("t1", "u2", broadcastID, "", 0)
	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Body != "Hello broadcast" {
		t.Fatalf("expected broadcast message, got %v", msgs)
	}
}
