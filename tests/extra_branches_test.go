package tests

import (
	"testing"

	chatroom "github.com/veltylabs/chat_room"
	"github.com/veltylabs/chat_room/migrate"
	"webtyp.com/fmt"
)

var _ = migrate.Migrate

func TestExtraValidationAndErrorBranches(t *testing.T) {
	partsList := []fmt.KeyValue{
		{Key: "u1", Value: "Alice"},
		{Key: "u2", Value: "Bob"},
	}
	env, err := setupTestEnv("t1", 30, partsList)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// 1. Unauthenticated or empty userID
	_, err = env.module.ListParticipants("t1", "")
	if err == nil || err.Error() != chatroom.ErrUnauthenticated.Error() {
		t.Fatalf("expected ErrUnauthenticated")
	}

	_, err = env.module.Heartbeat("t1", "")
	if err == nil || err.Error() != chatroom.ErrUnauthenticated.Error() {
		t.Fatalf("expected ErrUnauthenticated")
	}

	_, err = env.module.OpenDirect("t1", "", "u2")
	if err == nil || err.Error() != chatroom.ErrUnauthenticated.Error() {
		t.Fatalf("expected ErrUnauthenticated")
	}

	_, err = env.module.ListRooms("t1", "")
	if err == nil || err.Error() != chatroom.ErrUnauthenticated.Error() {
		t.Fatalf("expected ErrUnauthenticated")
	}

	_, err = env.module.ListMessages("t1", "", "r1", "", 0)
	if err == nil || err.Error() != chatroom.ErrUnauthenticated.Error() {
		t.Fatalf("expected ErrUnauthenticated")
	}

	_, err = env.module.SendMessage("t1", "", "r1", "msg")
	if err == nil || err.Error() != chatroom.ErrUnauthenticated.Error() {
		t.Fatalf("expected ErrUnauthenticated")
	}

	err = env.module.MarkRead("t1", "", "r1")
	if err == nil || err.Error() != chatroom.ErrUnauthenticated.Error() {
		t.Fatalf("expected ErrUnauthenticated")
	}

	// 2. Non-existent group
	_, err = env.module.SaveGroup("t1", "non-existent-id", "New Name")
	if err == nil || err.Error() != chatroom.ErrNotFound.Error() {
		t.Fatalf("expected ErrNotFound for non-existent SaveGroup, got %v", err)
	}

	err = env.module.SetGroupMembers("t1", "non-existent-id", []string{"u1"})
	if err == nil || err.Error() != chatroom.ErrNotFound.Error() {
		t.Fatalf("expected ErrNotFound for non-existent SetGroupMembers, got %v", err)
	}

	_, err = env.module.ListGroupMembers("t1", "non-existent-id")
	if err == nil || err.Error() != chatroom.ErrNotFound.Error() {
		t.Fatalf("expected ErrNotFound for non-existent ListGroupMembers, got %v", err)
	}

	err = env.module.DeleteGroup("t1", "non-existent-id")
	if err == nil || err.Error() != chatroom.ErrNotFound.Error() {
		t.Fatalf("expected ErrNotFound for non-existent DeleteGroup, got %v", err)
	}

	// 3. Operating on Direct Room when Group is expected
	directRoom, err := env.module.OpenDirect("t1", "u1", "u2")
	if err != nil {
		t.Fatalf("OpenDirect failed: %v", err)
	}

	_, err = env.module.SaveGroup("t1", directRoom.RoomId, "Try Rename Direct")
	if err == nil || err.Error() != chatroom.ErrNotGroup.Error() {
		t.Fatalf("expected ErrNotGroup for SaveGroup on direct room, got %v", err)
	}

	err = env.module.SetGroupMembers("t1", directRoom.RoomId, []string{"u1"})
	if err == nil || err.Error() != chatroom.ErrNotGroup.Error() {
		t.Fatalf("expected ErrNotGroup for SetGroupMembers on direct room, got %v", err)
	}

	_, err = env.module.ListGroupMembers("t1", directRoom.RoomId)
	if err == nil || err.Error() != chatroom.ErrNotGroup.Error() {
		t.Fatalf("expected ErrNotGroup for ListGroupMembers on direct room, got %v", err)
	}

	err = env.module.DeleteGroup("t1", directRoom.RoomId)
	if err == nil || err.Error() != chatroom.ErrNotGroup.Error() {
		t.Fatalf("expected ErrNotGroup for DeleteGroup on direct room, got %v", err)
	}

	// 4. Unknown user in Heartbeat
	_, err = env.module.Heartbeat("t1", "u_unknown")
	if err == nil || err.Error() != chatroom.ErrUnknownUser.Error() {
		t.Fatalf("expected ErrUnknownUser for Heartbeat")
	}

	// 5. SaveGroup name length validation
	_, err = env.module.SaveGroup("t1", "", "")
	if err == nil || err.Error() != chatroom.ErrNotFound.Error() {
		t.Fatalf("expected ErrNotFound for empty group name")
	}

	// 6. SetGroupMembers unknown user
	group, err := env.module.SaveGroup("t1", "", "Valid Group")
	if err != nil {
		t.Fatalf("SaveGroup failed: %v", err)
	}

	err = env.module.SetGroupMembers("t1", group.Id, []string{"u_unknown"})
	if err == nil || err.Error() != chatroom.ErrUnknownUser.Error() {
		t.Fatalf("expected ErrUnknownUser for SetGroupMembers")
	}

	// 7. Non-existent room in ListMessages / SendMessage / MarkRead
	_, err = env.module.ListMessages("t1", "u1", "room_unknown", "", 0)
	if err == nil || err.Error() != chatroom.ErrNotFound.Error() {
		t.Fatalf("expected ErrNotFound for ListMessages room_unknown")
	}

	_, err = env.module.SendMessage("t1", "u1", "room_unknown", "body")
	if err == nil || err.Error() != chatroom.ErrNotFound.Error() {
		t.Fatalf("expected ErrNotFound for SendMessage room_unknown")
	}

	err = env.module.MarkRead("t1", "u1", "room_unknown")
	if err == nil || err.Error() != chatroom.ErrNotFound.Error() {
		t.Fatalf("expected ErrNotFound for MarkRead room_unknown")
	}

	// 8. after_id not found in ListMessages
	_, err = env.module.ListMessages("t1", "u1", directRoom.RoomId, "after_unknown", 0)
	if err == nil || err.Error() != chatroom.ErrNotFound.Error() {
		t.Fatalf("expected ErrNotFound for ListMessages after_unknown")
	}
}
