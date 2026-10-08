package tests

import (
	"testing"

	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/fmt"
)

func TestSendMessageValidation(t *testing.T) {
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

	// Empty body
	_, err = env.module.SendMessage("t1", "u1", room.RoomId, "")
	if err == nil || err.Error() != chatroom.ErrEmptyBody.Error() {
		t.Fatalf("expected ErrEmptyBody, got %v", err)
	}

	// Only spaces
	_, err = env.module.SendMessage("t1", "u1", room.RoomId, "   \n\t  ")
	if err == nil || err.Error() != chatroom.ErrEmptyBody.Error() {
		t.Fatalf("expected ErrEmptyBody, got %v", err)
	}

	// Exceeds MaxBodyLength (2000 runes)
	tooLongRunes := make([]rune, chatroom.MaxBodyLength+1)
	for i := range tooLongRunes {
		tooLongRunes[i] = 'a'
	}
	_, err = env.module.SendMessage("t1", "u1", room.RoomId, string(tooLongRunes))
	if err == nil || err.Error() != chatroom.ErrBodyTooLong.Error() {
		t.Fatalf("expected ErrBodyTooLong, got %v", err)
	}

	// Valid message assigns sender_label from ParticipantReader
	msg, err := env.module.SendMessage("t1", "u1", room.RoomId, "Valid message")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if msg.SenderLabel != "Alice" {
		t.Fatalf("expected SenderLabel 'Alice', got '%s'", msg.SenderLabel)
	}
}

func TestMessagePagination(t *testing.T) {
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

	var msgIDs []string
	for i := 1; i <= 5; i++ {
		env.clockTime += 1000
		m, err := env.module.SendMessage("t1", "u1", room.RoomId, fmt.Sprintf("Msg %d", i))
		if err != nil {
			t.Fatalf("SendMessage %d failed: %v", i, err)
		}
		msgIDs = append(msgIDs, m.Id)
	}

	// after_id limit: querying after msg 1 with limit 2 returns msg 2 and msg 3
	msgs, err := env.module.ListMessages("t1", "u1", room.RoomId, msgIDs[0], 2)
	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Id != msgIDs[1] || msgs[1].Id != msgIDs[2] {
		t.Fatalf("unexpected paginated result: %v", msgs)
	}
}
