package tests

import (
	"testing"

	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/fmt"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
)

func TestNewDependenciesValidation(t *testing.T) {
	db := orm.New(mem.New())
	idGen := &mockIDGen{}
	parts := &mockParticipants{list: []fmt.KeyValue{{Key: "u1", Value: "User 1"}}}

	// Missing IDs
	_, err := chatroom.New(db, chatroom.Deps{
		Participants:  parts,
		TenantID:      "t1",
		RetentionDays: 30,
	})
	if err == nil || err.Error() != "chat_room: Deps.IDs is required" {
		t.Fatalf("expected Deps.IDs is required, got %v", err)
	}

	// Missing Participants
	_, err = chatroom.New(db, chatroom.Deps{
		IDs:           idGen,
		TenantID:      "t1",
		RetentionDays: 30,
	})
	if err == nil || err.Error() != "chat_room: Deps.Participants is required" {
		t.Fatalf("expected Deps.Participants is required, got %v", err)
	}

	// Missing TenantID
	_, err = chatroom.New(db, chatroom.Deps{
		IDs:           idGen,
		Participants:  parts,
		RetentionDays: 30,
	})
	if err == nil || err.Error() != "chat_room: Deps.TenantID is required" {
		t.Fatalf("expected Deps.TenantID is required, got %v", err)
	}

	// RetentionDays <= 0
	_, err = chatroom.New(db, chatroom.Deps{
		IDs:           idGen,
		Participants:  parts,
		TenantID:      "t1",
		RetentionDays: 0,
	})
	if err == nil || err.Error() != "chat_room: Deps.RetentionDays must be greater than zero" {
		t.Fatalf("expected Deps.RetentionDays must be greater than zero, got %v", err)
	}
}
