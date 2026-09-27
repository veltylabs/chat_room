package tests

import (
	"testing"

	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/events"
	"webtyp.com/fmt"
)

func TestInboxEvents(t *testing.T) {
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

	var sentEvents []events.Event
	var u1InboxCount int
	var u2InboxCount int

	env.broker.Subscribe(chatroom.TopicMessageSent, func(e events.Event) {
		sentEvents = append(sentEvents, e)
	})
	env.broker.Subscribe(chatroom.InboxTopic("u1"), func(e events.Event) {
		u1InboxCount++
	})
	env.broker.Subscribe(chatroom.InboxTopic("u2"), func(e events.Event) {
		u2InboxCount++
	})

	// Send message in direct u1 -> u2
	_, err = env.module.SendMessage("t1", "u1", room.RoomId, "Direct notification test")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	// Verify TopicMessageSent event
	if len(sentEvents) != 1 {
		t.Fatalf("expected 1 TopicMessageSent event, got %d", len(sentEvents))
	}

	// Verify InboxTopic(u2) event published, and NOT InboxTopic(u1)
	if u2InboxCount != 1 {
		t.Fatalf("expected 1 InboxTopic(u2) event, got %d", u2InboxCount)
	}
	if u1InboxCount != 0 {
		t.Fatalf("expected 0 InboxTopic(u1) events, got %d", u1InboxCount)
	}

	// Verify no event carries the message body in TopicMessageSent
	evtPayload, ok := sentEvents[0].Payload.(*chatroom.MessageSentEvent)
	if !ok {
		t.Fatalf("expected *chatroom.MessageSentEvent payload")
	}
	if evtPayload.MessageId == "" || evtPayload.SenderId != "u1" {
		t.Fatalf("unexpected message sent payload: %v", evtPayload)
	}
}
