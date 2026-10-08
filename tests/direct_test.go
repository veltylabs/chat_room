package tests

import (
	"testing"

	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/fmt"
)

func TestOpenDirect(t *testing.T) {
	partsList := []fmt.KeyValue{
		{Key: "u1", Value: "Alice"},
		{Key: "u2", Value: "Bob"},
		{Key: "u3", Value: "Charlie"},
	}
	env, err := setupTestEnv("t1", 30, partsList)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// 1. OpenDirect a -> b and b -> a yields SAME room
	room1, err := env.module.OpenDirect("t1", "u1", "u2")
	if err != nil {
		t.Fatalf("OpenDirect u1->u2 failed: %v", err)
	}

	room2, err := env.module.OpenDirect("t1", "u2", "u1")
	if err != nil {
		t.Fatalf("OpenDirect u2->u1 failed: %v", err)
	}

	if room1.RoomId != room2.RoomId {
		t.Fatalf("expected same room ID, got %s vs %s", room1.RoomId, room2.RoomId)
	}

	// 2. OpenDirect with self -> ErrSelfDirect
	_, err = env.module.OpenDirect("t1", "u1", "u1")
	if err == nil || err.Error() != chatroom.ErrSelfDirect.Error() {
		t.Fatalf("expected ErrSelfDirect, got %v", err)
	}

	// 3. OpenDirect with unknown user -> ErrUnknownUser
	_, err = env.module.OpenDirect("t1", "u1", "u_unknown")
	if err == nil || err.Error() != chatroom.ErrUnknownUser.Error() {
		t.Fatalf("expected ErrUnknownUser, got %v", err)
	}
}

func TestDirectPrivacy(t *testing.T) {
	partsList := []fmt.KeyValue{
		{Key: "u1", Value: "Alice"},
		{Key: "u2", Value: "Bob"},
		{Key: "u3", Value: "Charlie"},
	}
	env, err := setupTestEnv("t1", 30, partsList)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	room, err := env.module.OpenDirect("t1", "u1", "u2")
	if err != nil {
		t.Fatalf("OpenDirect failed: %v", err)
	}

	_, err = env.module.SendMessage("t1", "u1", room.RoomId, "Secret direct message")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	// Third party u3 cannot ListMessages
	_, err = env.module.ListMessages("t1", "u3", room.RoomId, "", 0)
	if err == nil || err.Error() != chatroom.ErrNotMember.Error() {
		t.Fatalf("expected ErrNotMember for u3 ListMessages, got %v", err)
	}

	// Third party u3 cannot SendMessage
	_, err = env.module.SendMessage("t1", "u3", room.RoomId, "Sneaky message")
	if err == nil || err.Error() != chatroom.ErrNotMember.Error() {
		t.Fatalf("expected ErrNotMember for u3 SendMessage, got %v", err)
	}

	// Third party u3 cannot MarkRead
	err = env.module.MarkRead("t1", "u3", room.RoomId)
	if err == nil || err.Error() != chatroom.ErrNotMember.Error() {
		t.Fatalf("expected ErrNotMember for u3 MarkRead, got %v", err)
	}
}
