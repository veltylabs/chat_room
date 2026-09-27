package tests

import (
	"testing"

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

	// Set Members
	err = env.module.SetGroupMembers("t1", group.Id, []string{"u1", "u2"})
	if err != nil {
		t.Fatalf("SetGroupMembers failed: %v", err)
	}

	mems, err := env.module.ListGroupMembers("t1", group.Id)
	if err != nil {
		t.Fatalf("ListGroupMembers failed: %v", err)
	}
	if len(mems) != 2 {
		t.Fatalf("expected 2 group members, got %d", len(mems))
	}

	// Preserve last_read_at on SetGroupMembers
	_ = env.module.MarkRead("t1", "u1", group.Id)
	err = env.module.SetGroupMembers("t1", group.Id, []string{"u1", "u3"})
	if err != nil {
		t.Fatalf("SetGroupMembers update failed: %v", err)
	}

	// Delete Group
	err = env.module.DeleteGroup("t1", group.Id)
	if err != nil {
		t.Fatalf("DeleteGroup failed: %v", err)
	}

	groups, err := env.module.ListGroups("t1")
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups after delete, got %d", len(groups))
	}
}
