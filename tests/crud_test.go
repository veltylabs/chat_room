package tests

import (
	"strings"
	"testing"

	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/crud"
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/router"
)

// call runs one operation of the module as `user` through the same loopback the demo uses and answers the callback error.
func call(user string, pub events.Publisher, mod *chatroom.Module, entity string, action model.Action, args model.Encodable, into model.Decodable) error {
	var callErr error
	crud.LoopbackActingAs(pub, user, mod).Call(
		router.Operation{Module: chatroom.ModelName, Entity: entity, Action: action},
		args, into, func(err error) { callErr = err })
	return callErr
}

// failedField answers the field a refusal points at, or "".
func failedField(err error) string {
	if f, ok := err.(interface{ FailedField() string }); ok {
		return f.FailedField()
	}
	return ""
}

func TestMessage_NonMemberRejected(t *testing.T) {
	env, err := setupTestEnv("t1", 30, []fmt.KeyValue{
		{Key: "u1", Value: "User 1"},
		{Key: "u2", Value: "User 2"},
		{Key: "u3", Value: "User 3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	group, err := env.module.SaveGroup("t1", "", "Team")
	if err != nil {
		t.Fatal(err)
	}
	if err := env.module.SetGroupMembers("t1", group.Id, []string{"u1", "u2"}); err != nil {
		t.Fatal(err)
	}

	err = call("u3", nil, env.module, "chat_message", model.Create, &crud.CreateArgs{
		Records: []model.Model{&chatroom.Message{Id: "m1", RoomId: group.Id, Body: "hi"}},
	}, nil)
	if err == nil {
		t.Fatalf("expected error for non-member create message, got nil")
	}

	msgs, err := env.module.ListMessages("t1", "u1", group.Id, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(msgs))
	}
}

func TestMessage_CreateSetsSenderFields(t *testing.T) {
	env, err := setupTestEnv("t1", 30, []fmt.KeyValue{
		{Key: "u1", Value: "User 1"},
		{Key: "u2", Value: "User 2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	group, err := env.module.SaveGroup("t1", "", "Team")
	if err != nil {
		t.Fatal(err)
	}
	if err := env.module.SetGroupMembers("t1", group.Id, []string{"u1", "u2"}); err != nil {
		t.Fatal(err)
	}

	err = call("u1", nil, env.module, "chat_message", model.Create, &crud.CreateArgs{
		Records: []model.Model{&chatroom.Message{Id: "m1", RoomId: group.Id, Body: "hi"}},
	}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var list chatroom.MessageViewList
	err = call("u1", nil, env.module, "message_view", model.Read, &crud.ReadArgs{
		Filter: &chatroom.MessageView{RoomId: group.Id},
	}, &list)
	if err != nil {
		t.Fatalf("expected no error reading message_view, got %v", err)
	}
	if list.Len() != 1 {
		t.Fatalf("expected 1 message, got %d", list.Len())
	}
	msg := list.At(0).(*chatroom.MessageView)
	if msg.SenderId != "u1" || !msg.Mine {
		t.Fatalf("expected SenderId==u1 and Mine==true, got %q, %v", msg.SenderId, msg.Mine)
	}
}

func TestMessages_NonMemberCannotRead(t *testing.T) {
	env, err := setupTestEnv("t1", 30, []fmt.KeyValue{
		{Key: "u1", Value: "User 1"},
		{Key: "u2", Value: "User 2"},
		{Key: "u3", Value: "User 3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	group, err := env.module.SaveGroup("t1", "", "Team")
	if err != nil {
		t.Fatal(err)
	}
	if err := env.module.SetGroupMembers("t1", group.Id, []string{"u1", "u2"}); err != nil {
		t.Fatal(err)
	}

	var list chatroom.MessageViewList
	err = call("u3", nil, env.module, "message_view", model.Read, &crud.ReadArgs{
		Filter: &chatroom.MessageView{RoomId: group.Id},
	}, &list)
	if err == nil || model.KindOf(err) != model.ErrorNotFound {
		t.Fatalf("expected ErrorNotFound, got %v", err)
	}
}

func TestMessages_ReadModelNeedsRoom(t *testing.T) {
	env, err := setupTestEnv("t1", 30, []fmt.KeyValue{
		{Key: "u1", Value: "User 1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var list chatroom.MessageViewList
	err = call("u1", nil, env.module, "message_view", model.Read, &crud.ReadArgs{
		Filter: &chatroom.MessageView{},
	}, &list)
	if err == nil || failedField(err) != "room_id" {
		t.Fatalf("expected failedField room_id, got %v", err)
	}
}

func TestMember_MarkReadOnlyOwnRow(t *testing.T) {
	env, err := setupTestEnv("t1", 30, []fmt.KeyValue{
		{Key: "u1", Value: "User 1"},
		{Key: "u2", Value: "User 2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	group, err := env.module.SaveGroup("t1", "", "Team")
	if err != nil {
		t.Fatal(err)
	}
	if err := env.module.SetGroupMembers("t1", group.Id, []string{"u1", "u2"}); err != nil {
		t.Fatal(err)
	}

	err = call("u1", nil, env.module, "chat_member", model.Update, &crud.UpdateArgs{
		IDs:    []string{model.JoinKey(group.Id, "u2")},
		Fields: []string{"last_read_at"},
		Record: &chatroom.Member{LastReadAt: 1},
	}, nil)
	if err == nil || failedField(err) != "user_id" {
		t.Fatalf("expected failedField user_id, got %v", err)
	}

	err = call("u1", nil, env.module, "chat_member", model.Update, &crud.UpdateArgs{
		IDs:    []string{model.JoinKey(group.Id, "u1")},
		Fields: []string{"last_read_at"},
		Record: &chatroom.Member{LastReadAt: 1},
	}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var memberList chatroom.MemberList
	err = call("u1", nil, env.module, "chat_member", model.Read, &crud.ReadArgs{
		ID: model.JoinKey(group.Id, "u1"),
	}, &memberList)
	if err != nil {
		t.Fatal(err)
	}
	if memberList.Len() != 1 || memberList.At(0).(*chatroom.Member).LastReadAt <= 1 {
		t.Fatalf("expected stored LastReadAt > 1, got %+v", memberList.At(0))
	}
}

func TestMember_UpdateOnlyLastRead(t *testing.T) {
	env, err := setupTestEnv("t1", 30, []fmt.KeyValue{
		{Key: "u1", Value: "User 1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	group, err := env.module.SaveGroup("t1", "", "Team")
	if err != nil {
		t.Fatal(err)
	}
	if err := env.module.SetGroupMembers("t1", group.Id, []string{"u1"}); err != nil {
		t.Fatal(err)
	}

	err = call("u1", nil, env.module, "chat_member", model.Update, &crud.UpdateArgs{
		IDs:    []string{model.JoinKey(group.Id, "u1")},
		Fields: []string{"room_id"},
		Record: &chatroom.Member{},
	}, nil)
	if err == nil || (failedField(err) != "room_id" && model.KindOf(err) != model.ErrorInvalid) {
		t.Fatalf("expected failedField room_id or ErrorInvalid, got %v", err)
	}
}

func TestMember_CreateNeedsKnownGroupUser(t *testing.T) {
	env, err := setupTestEnv("t1", 30, []fmt.KeyValue{
		{Key: "u1", Value: "User 1"},
		{Key: "u3", Value: "User 3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	group, err := env.module.SaveGroup("t1", "", "Team")
	if err != nil {
		t.Fatal(err)
	}

	err = call("u1", nil, env.module, "chat_member", model.Create, &crud.CreateArgs{
		Records: []model.Model{&chatroom.Member{RoomId: group.Id, UserId: "nobody"}},
	}, nil)
	if err == nil || failedField(err) != "user_id" {
		t.Fatalf("expected failedField user_id, got %v", err)
	}

	err = call("u1", nil, env.module, "chat_member", model.Create, &crud.CreateArgs{
		Records: []model.Model{&chatroom.Member{RoomId: group.Id, UserId: "u3"}},
	}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = call("u1", nil, env.module, "chat_member", model.Create, &crud.CreateArgs{
		Records: []model.Model{&chatroom.Member{RoomId: group.Id, UserId: "u3"}},
	}, nil)
	if err != nil && model.KindOf(err) != model.ErrorConflict {
		t.Fatalf("expected nil or ErrorConflict, got %v", err)
	}
}

func TestDirect_SamePairOnce(t *testing.T) {
	env, err := setupTestEnv("t1", 30, []fmt.KeyValue{
		{Key: "u1", Value: "User 1"},
		{Key: "u2", Value: "User 2"},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = call("u1", nil, env.module, "chat_direct", model.Create, &crud.CreateArgs{
		Records: []model.Model{&chatroom.Direct{UserA: "u1", UserB: "u2"}},
	}, nil)
	if err != nil {
		t.Fatalf("expected no error creating direct, got %v", err)
	}

	err = call("u1", nil, env.module, "chat_direct", model.Create, &crud.CreateArgs{
		Records: []model.Model{&chatroom.Direct{UserA: "u1", UserB: "u2"}},
	}, nil)
	if err != nil && model.KindOf(err) != model.ErrorConflict {
		t.Fatalf("expected nil or ErrorConflict, got %v", err)
	}

	summary, err := env.module.OpenDirect("", "u2", "u1")
	if err != nil {
		t.Fatalf("OpenDirect failed: %v", err)
	}

	var directList chatroom.DirectList
	err = call("u1", nil, env.module, "chat_direct", model.Read, &crud.ReadArgs{
		ID: model.JoinKey("u1", "u2"),
	}, &directList)
	if err != nil || directList.Len() != 1 {
		t.Fatalf("read direct failed: %v", err)
	}
	if directList.At(0).(*chatroom.Direct).RoomId != summary.RoomId {
		t.Fatalf("expected room ID %q, got %q", summary.RoomId, directList.At(0).(*chatroom.Direct).RoomId)
	}
}

func TestDirect_CallerMustBeInPair(t *testing.T) {
	env, err := setupTestEnv("t1", 30, []fmt.KeyValue{
		{Key: "u1", Value: "User 1"},
		{Key: "u2", Value: "User 2"},
		{Key: "u3", Value: "User 3"},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = call("u3", nil, env.module, "chat_direct", model.Create, &crud.CreateArgs{
		Records: []model.Model{&chatroom.Direct{UserA: "u1", UserB: "u2"}},
	}, nil)
	if err == nil || failedField(err) != "user_a" {
		t.Fatalf("expected failedField user_a, got %v", err)
	}
}

func TestDirect_PairMustBeSorted(t *testing.T) {
	env, err := setupTestEnv("t1", 30, []fmt.KeyValue{
		{Key: "u1", Value: "User 1"},
		{Key: "u2", Value: "User 2"},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = call("u2", nil, env.module, "chat_direct", model.Create, &crud.CreateArgs{
		Records: []model.Model{&chatroom.Direct{UserA: "u2", UserB: "u1"}},
	}, nil)
	if err == nil || failedField(err) != "user_a" {
		t.Fatalf("expected failedField user_a, got %v", err)
	}
}

func TestPresence_OnlyOwn(t *testing.T) {
	env, err := setupTestEnv("t1", 30, []fmt.KeyValue{
		{Key: "u1", Value: "User 1"},
		{Key: "u2", Value: "User 2"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed presence for u2 so the record exists before u1 attempts to update it.
	err = call("u2", nil, env.module, "chat_presence", model.Create, &crud.CreateArgs{
		Records: []model.Model{&chatroom.Presence{UserId: "u2", LastSeen: 1}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = call("u1", nil, env.module, "chat_presence", model.Update, &crud.UpdateArgs{
		IDs:    []string{"u2"},
		Fields: []string{"last_seen"},
		Record: &chatroom.Presence{LastSeen: 1},
	}, nil)
	if err == nil || failedField(err) != "user_id" {
		t.Fatalf("expected failedField user_id, got %v", err)
	}
}

func TestGroup_DeleteCascades(t *testing.T) {
	env, err := setupTestEnv("t1", 30, []fmt.KeyValue{
		{Key: "u1", Value: "User 1"},
		{Key: "u2", Value: "User 2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	group, err := env.module.SaveGroup("t1", "", "Team")
	if err != nil {
		t.Fatal(err)
	}
	if err := env.module.SetGroupMembers("t1", group.Id, []string{"u1", "u2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.module.SendMessage("t1", "u1", group.Id, "msg 1"); err != nil {
		t.Fatal(err)
	}
	if _, err := env.module.SendMessage("t1", "u2", group.Id, "msg 2"); err != nil {
		t.Fatal(err)
	}
	if _, err := env.module.SendMessage("t1", "u1", group.Id, "msg 3"); err != nil {
		t.Fatal(err)
	}

	err = call("u1", nil, env.module, "chat_room", model.Delete, &crud.DeleteArgs{
		IDs: []string{group.Id},
	}, nil)
	if err != nil {
		t.Fatalf("expected delete to succeed, got %v", err)
	}

	var memberList chatroom.MemberList
	err = call("u1", nil, env.module, "chat_member", model.Read, &crud.ReadArgs{
		Filter: &chatroom.Member{RoomId: group.Id},
	}, &memberList)
	if err != nil {
		t.Fatal(err)
	}
	if memberList.Len() != 0 {
		t.Fatalf("expected 0 members, got %d", memberList.Len())
	}

	_, err = env.module.ListMessages("t1", "u1", group.Id, "", 0)
	if err == nil || err.Error() != chatroom.ErrNotFound.Error() {
		t.Fatalf("expected ErrNotFound for deleted room ListMessages, got %v", err)
	}
}

func TestGroup_DeleteRejectsNonGroup(t *testing.T) {
	env, err := setupTestEnv("t1", 30, []fmt.KeyValue{
		{Key: "u1", Value: "User 1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rooms, err := env.module.ListRooms("t1", "u1")
	if err != nil || len(rooms) == 0 {
		t.Fatal(err)
	}
	broadcastID := rooms[0].RoomId

	err = call("u1", nil, env.module, "chat_room", model.Delete, &crud.DeleteArgs{
		IDs: []string{broadcastID},
	}, nil)
	if err == nil || !strings.Contains(err.Error(), string(chatroom.ErrNotGroup)) {
		t.Fatalf("expected ErrNotGroup in error, got %v", err)
	}
}

func TestMount_MatchesDefinition(t *testing.T) {
	env, err := setupTestEnv("t1", 30, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := crud.Mount(nil, nil, nil, env.module); err != nil {
		t.Fatalf("crud.Mount failed: %v", err)
	}
}

type testPub struct {
	published []events.Event
}

func (p *testPub) Publish(e events.Event) {
	p.published = append(p.published, e)
}

func TestCreate_PublishesChange(t *testing.T) {
	env, err := setupTestEnv("t1", 30, nil)
	if err != nil {
		t.Fatal(err)
	}
	pub := &testPub{}
	err = call("u1", pub, env.module, "chat_room", model.Create, &crud.CreateArgs{
		Records: []model.Model{&chatroom.Room{Id: "g1", Kind: chatroom.KindGroup, Name: "Test Group"}},
	}, nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	wantTopic := crud.Topic(chatroom.ModelName, (&chatroom.Room{}).ModelName(), model.Create)
	var found bool
	for _, e := range pub.published {
		if e.Topic == wantTopic {
			found = true
			if _, ok := e.Payload.(*crud.Change); !ok {
				t.Fatalf("expected payload *crud.Change, got %T", e.Payload)
			}
		}
	}
	if !found {
		t.Fatalf("expected event on topic %q, got %+v", wantTopic, pub.published)
	}
}

func TestDefaultGrants_SignedInCanSend(t *testing.T) {
	var foundMessageCreate bool
	for _, g := range chatroom.DefaultGrants {
		if g.Resource == model.ResourceOf(&chatroom.MessageModel) && (g.Actions&model.Create != 0) {
			foundMessageCreate = true
		}
		if g.Actions&model.Delete != 0 {
			t.Fatalf("expected no DefaultGrant with model.Delete, got %+v", g)
		}
	}
	if !foundMessageCreate {
		t.Fatalf("expected DefaultGrants to contain MessageModel with model.Create")
	}
}
