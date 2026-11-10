package tests

import (
	"testing"

	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/fmt"
)

func TestGroupManagement(t *testing.T) {
	partsList := []fmt.KeyValue{
		{Key: "u1", Value: "Alice"},
		{Key: "u2", Value: "Bob"},
		{Key: "u3", Value: "Charlie"},
	}
	env, err := setupTestEnv("t1", 30, partsList)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Create Group
	group, err := env.module.SaveGroup("t1", "", "Team Alpha")
	if err != nil {
		t.Fatalf("SaveGroup failed: %v", err)
	}
	if group.Kind != chatroom.KindGroup || group.Name != "Team Alpha" {
		t.Fatalf("unexpected group: %+v", group)
	}

	// Rename Group
	renamed, err := env.module.SaveGroup("t1", group.Id, "Team Bravo")
	if err != nil {
		t.Fatalf("SaveGroup (rename) failed: %v", err)
	}
	if renamed.Name != "Team Bravo" {
		t.Fatalf("expected Team Bravo, got %s", renamed.Name)
	}

	// Set Members
	err = env.module.SetGroupMembers("t1", group.Id, []string{"u1", "u2"})
	if err != nil {
		t.Fatalf("SetGroupMembers failed: %v", err)
	}

	members, err := env.module.ListGroupMembers("t1", group.Id)
	if err != nil {
		t.Fatalf("ListGroupMembers failed: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}

	// List Groups
	groups, err := env.module.ListGroups("", "", 0)
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}
	if len(groups) != 1 || groups[0].Name != "Team Bravo" {
		t.Fatalf("unexpected groups: %+v", groups)
	}

	// Delete Group
	err = env.module.DeleteGroup("t1", group.Id)
	if err != nil {
		t.Fatalf("DeleteGroup failed: %v", err)
	}

	groups, err = env.module.ListGroups("", "", 0)
	if err != nil {
		t.Fatalf("ListGroups after delete failed: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups after delete, got %d", len(groups))
	}
}
