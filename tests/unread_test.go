package tests

import (
	"testing"

	"webtyp.com/fmt"
)

func TestUnreadAndReadReceipts(t *testing.T) {
	partsList := []fmt.KeyValue{
		{Key: "u1", Value: "Alice"},
		{Key: "u2", Value: "Bob"},
	}
	env, err := setupTestEnv("t1", 30, partsList)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	room, err := env.module.OpenDirect("t1", "u1", "u2")
	if err != nil {
		t.Fatalf("OpenDirect failed: %v", err)
	}

	// u1 sends 2 messages to u2
	env.clockTime += 1000
	m1, err := env.module.SendMessage("t1", "u1", room.RoomId, "Message 1")
	if err != nil {
		t.Fatalf("SendMessage 1 failed: %v", err)
	}
	env.clockTime += 1000
	_, err = env.module.SendMessage("t1", "u1", room.RoomId, "Message 2")
	if err != nil {
		t.Fatalf("SendMessage 2 failed: %v", err)
	}

	// Check u2 unread count = 2
	roomsU2, err := env.module.ListRooms("t1", "u2")
	if err != nil {
		t.Fatalf("ListRooms u2 failed: %v", err)
	}
	var u2SummaryRoom *struct{ Unread int64 }
	for _, r := range roomsU2 {
		if r.RoomId == room.RoomId {
			u2SummaryRoom = &struct{ Unread int64 }{Unread: r.Unread}
			break
		}
	}
	if u2SummaryRoom == nil || u2SummaryRoom.Unread != 2 {
		t.Fatalf("expected 2 unread messages for u2, got %v", u2SummaryRoom)
	}

	// Read status in direct before u2 MarkRead: mine=true, read=false
	msgsU1, err := env.module.ListMessages("t1", "u1", room.RoomId, "", 0)
	if err != nil {
		t.Fatalf("ListMessages u1 failed: %v", err)
	}
	if !msgsU1[0].Mine || msgsU1[0].Read {
		t.Fatalf("expected mine=true, read=false before u2 reads, got mine=%v, read=%v", msgsU1[0].Mine, msgsU1[0].Read)
	}

	// u2 calls MarkRead
	env.clockTime += 1000
	err = env.module.MarkRead("t1", "u2", room.RoomId)
	if err != nil {
		t.Fatalf("MarkRead u2 failed: %v", err)
	}

	// Check u2 unread count = 0
	roomsU2After, err := env.module.ListRooms("t1", "u2")
	if err != nil {
		t.Fatalf("ListRooms u2 after MarkRead failed: %v", err)
	}
	for _, r := range roomsU2After {
		if r.RoomId == room.RoomId && r.Unread != 0 {
			t.Fatalf("expected 0 unread for u2 after MarkRead, got %d", r.Unread)
		}
	}

	// Read status in direct after u2 MarkRead: read=true for m1
	msgsU1After, err := env.module.ListMessages("t1", "u1", room.RoomId, "", 0)
	if err != nil {
		t.Fatalf("ListMessages u1 after MarkRead failed: %v", err)
	}
	if msgsU1After[0].Id == m1.Id && !msgsU1After[0].Read {
		t.Fatalf("expected m1 read=true after u2 MarkRead")
	}
}
