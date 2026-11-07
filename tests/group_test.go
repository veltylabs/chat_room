package tests

import (
	"testing"

	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/router/mock"
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

	groups, err := env.module.ListGroups("", "", 500)
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups after delete, got %d", len(groups))
	}
}

func TestGroup_ListAcceptsListArgs(t *testing.T) {
	partsList := []fmt.KeyValue{
		{Key: "u1", Value: "Alice"},
	}
	env, err := setupTestEnv("t1", 30, partsList)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	g1, err := env.module.SaveGroup("t1", "", "Team Alpha")
	if err != nil { t.Fatalf("SaveGroup failed: %v", err) }
	_, err = env.module.SaveGroup("t1", "", "Team Beta")
	if err != nil { t.Fatalf("SaveGroup failed: %v", err) }

	r := &mock.Router{}
	r.Configure(mock.Config{
		Authorize: func(userID string, res model.Resource, a model.Action) bool {
			return true
		},
	})
	env.module.MountOperations(r)

	// Limit: 1
	ctx1 := &mock.Context{InBody: []byte(`{"limit": 1}`)}
	ctx1.SetUserID("u1")
	r.Invoke("OP", "/"+chatroom.OpListGroups, ctx1)
	if ctx1.Status != 200 && ctx1.Status != 0 {
		t.Fatalf("Limit: 1 failed: %d", ctx1.Status)
	}
	// Search: "Beta"
	ctx2 := &mock.Context{InBody: []byte(`{"search": "Beta"}`)}
	ctx2.SetUserID("u1")
	r.Invoke("OP", "/"+chatroom.OpListGroups, ctx2)
	if ctx2.Status != 200 && ctx2.Status != 0 {
		t.Fatalf("Search failed: %d", ctx2.Status)
	}
	// After: g1.Id
	ctx3 := &mock.Context{InBody: []byte(fmt.Sprintf(`{"after": "%s"}`, g1.Id))}
	ctx3.SetUserID("u1")
	r.Invoke("OP", "/"+chatroom.OpListGroups, ctx3)
	if ctx3.Status != 200 && ctx3.Status != 0 {
		t.Fatalf("After failed: %d", ctx3.Status)
	}
}
