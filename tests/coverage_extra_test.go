package tests

import (
	"testing"

	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/router/mock"
)

type mockCaller struct{}

func (m *mockCaller) Call(op string, args model.Encodable, into model.Decodable, done func(err error)) {
	if done != nil {
		done(nil)
	}
}

func (m *mockCaller) Dispatch(op string, args model.Encodable) {}

func TestAllOpsAndViews(t *testing.T) {
	partsList := []fmt.KeyValue{
		{Key: "u1", Value: "Alice"},
		{Key: "u2", Value: "Bob"},
	}
	env, err := setupTestEnv("t1", 30, partsList)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	r := &mock.Router{}
	r.Configure(mock.Config{
		Authorize: func(userID string, r model.Resource, a model.Action) bool {
			return true
		},
	})
	env.module.MountOperations(r)

	// ListParticipants Op
	ctx := &mock.Context{InBody: []byte(`{"tenant_id":"t1"}`)}
	ctx.SetUserID("u1")
	r.Invoke("OP", "/"+chatroom.OpListParticipants, ctx)
	if ctx.Status != 200 && ctx.Status != 0 {
		t.Fatalf("OpListParticipants failed: %d", ctx.Status)
	}

	// Heartbeat Op
	ctx = &mock.Context{InBody: []byte(`{"tenant_id":"t1"}`)}
	ctx.SetUserID("u1")
	r.Invoke("OP", "/"+chatroom.OpHeartbeat, ctx)
	if ctx.Status != 200 && ctx.Status != 0 {
		t.Fatalf("OpHeartbeat failed: %d", ctx.Status)
	}

	// OpenDirect Op
	ctx = &mock.Context{InBody: []byte(`{"tenant_id":"t1","other_user_id":"u2"}`)}
	ctx.SetUserID("u1")
	r.Invoke("OP", "/"+chatroom.OpOpenDirect, ctx)
	if ctx.Status != 200 && ctx.Status != 0 {
		t.Fatalf("OpOpenDirect failed: %d", ctx.Status)
	}

	// Direct room ID from env
	directSummary, err := env.module.OpenDirect("t1", "u1", "u2")
	if err != nil {
		t.Fatalf("OpenDirect failed: %v", err)
	}

	// SendMessage Op
	ctx = &mock.Context{InBody: []byte(fmt.Sprintf(`{"tenant_id":"t1","room_id":"%s","body":"Op Message"}`, directSummary.RoomId))}
	ctx.SetUserID("u1")
	r.Invoke("OP", "/"+chatroom.OpSendMessage, ctx)
	if ctx.Status != 200 && ctx.Status != 0 {
		t.Fatalf("OpSendMessage failed: %d", ctx.Status)
	}

	// ListMessages Op
	ctx = &mock.Context{InBody: []byte(fmt.Sprintf(`{"tenant_id":"t1","room_id":"%s"}`, directSummary.RoomId))}
	ctx.SetUserID("u1")
	r.Invoke("OP", "/"+chatroom.OpListMessages, ctx)
	if ctx.Status != 200 && ctx.Status != 0 {
		t.Fatalf("OpListMessages failed: %d", ctx.Status)
	}

	// MarkRead Op
	ctx = &mock.Context{InBody: []byte(fmt.Sprintf(`{"tenant_id":"t1","room_id":"%s"}`, directSummary.RoomId))}
	ctx.SetUserID("u2")
	r.Invoke("OP", "/"+chatroom.OpMarkRead, ctx)
	if ctx.Status != 200 && ctx.Status != 0 {
		t.Fatalf("OpMarkRead failed: %d", ctx.Status)
	}

	// SaveGroup Op
	ctx = &mock.Context{InBody: []byte(`{"tenant_id":"t1","name":"Op Group"}`)}
	ctx.SetUserID("u1")
	r.Invoke("OP", "/"+chatroom.OpSaveGroup, ctx)
	if ctx.Status != 200 && ctx.Status != 0 {
		t.Fatalf("OpSaveGroup failed: %d", ctx.Status)
	}

	// Group from env
	group, err := env.module.SaveGroup("t1", "", "Op Group")
	if err != nil {
		t.Fatalf("SaveGroup failed: %v", err)
	}

	// ListGroups Op
	ctx = &mock.Context{InBody: []byte(`{"tenant_id":"t1"}`)}
	ctx.SetUserID("u1")
	r.Invoke("OP", "/"+chatroom.OpListGroups, ctx)
	if ctx.Status != 200 && ctx.Status != 0 {
		t.Fatalf("OpListGroups failed: %d", ctx.Status)
	}

	// SetGroupMembers Op
	ctx = &mock.Context{InBody: []byte(fmt.Sprintf(`{"tenant_id":"t1","room_id":"%s","user_ids":[{"id":"u1"},{"id":"u2"}]}`, group.Id))}
	ctx.SetUserID("u1")
	r.Invoke("OP", "/"+chatroom.OpSetGroupMembers, ctx)
	if ctx.Status != 200 && ctx.Status != 0 {
		t.Fatalf("OpSetGroupMembers failed: %d", ctx.Status)
	}

	// ListGroupMembers Op
	ctx = &mock.Context{InBody: []byte(fmt.Sprintf(`{"tenant_id":"t1","room_id":"%s"}`, group.Id))}
	ctx.SetUserID("u1")
	r.Invoke("OP", "/"+chatroom.OpListGroupMembers, ctx)
	if ctx.Status != 200 && ctx.Status != 0 {
		t.Fatalf("OpListGroupMembers failed: %d", ctx.Status)
	}

	// DeleteGroup Op
	ctx = &mock.Context{InBody: []byte(fmt.Sprintf(`{"tenant_id":"t1","id":"%s"}`, group.Id))}
	ctx.SetUserID("u1")
	r.Invoke("OP", "/"+chatroom.OpDeleteGroup, ctx)
	if ctx.Status != 200 && ctx.Status != 0 {
		t.Fatalf("OpDeleteGroup failed: %d", ctx.Status)
	}

	// Exercise View Item and NewGroupView
	item := group.Item()
	if item.ID != group.Id {
		t.Fatalf("Item() failed")
	}

	p := chatroom.NewGroupView(&mockCaller{})
	if p == nil {
		t.Fatalf("NewGroupView returned nil")
	}
}

func TestModelORMMethods(t *testing.T) {
	r := &chatroom.Room{Id: "r1", TenantId: "t1", Kind: chatroom.KindGroup, Name: "Test"}
	if r.ModelName() != chatroom.ModelName {
		t.Fatalf("ModelName failed")
	}
	if r.Schema() == nil {
		t.Fatalf("Schema failed")
	}
	if len(r.Pointers()) == 0 {
		t.Fatalf("Pointers failed")
	}
	if r.IsNil() {
		t.Fatalf("IsNil failed")
	}

	var rList chatroom.RoomList
	rList = append(rList, r)
	if rList.Len() != 1 || rList.At(0) == nil || rList.IsNil() {
		t.Fatalf("RoomList methods failed")
	}

	m := &chatroom.Member{TenantId: "t1", RoomId: "r1", UserId: "u1"}
	if m.ModelName() == "" || m.Schema() == nil || len(m.Pointers()) == 0 || m.IsNil() {
		t.Fatalf("Member methods failed")
	}
	var mList chatroom.MemberList
	mList = append(mList, m)
	if mList.Len() != 1 || mList.At(0) == nil || mList.IsNil() {
		t.Fatalf("MemberList methods failed")
	}

	msg := &chatroom.Message{Id: "m1", TenantId: "t1", RoomId: "r1", SenderId: "u1", Body: "hi"}
	if msg.ModelName() == "" || msg.Schema() == nil || len(msg.Pointers()) == 0 || msg.IsNil() {
		t.Fatalf("Message methods failed")
	}
	var msgList chatroom.MessageList
	msgList = append(msgList, msg)
	if msgList.Len() != 1 || msgList.At(0) == nil || msgList.IsNil() {
		t.Fatalf("MessageList methods failed")
	}

	p := &chatroom.Presence{TenantId: "t1", UserId: "u1"}
	if p.ModelName() == "" || p.Schema() == nil || len(p.Pointers()) == 0 || p.IsNil() {
		t.Fatalf("Presence methods failed")
	}
	var pList chatroom.PresenceList
	pList = append(pList, p)
	if pList.Len() != 1 || pList.At(0) == nil || pList.IsNil() {
		t.Fatalf("PresenceList methods failed")
	}

	idRef := &chatroom.IdRef{Id: "id1"}
	if idRef.ModelName() == "" || idRef.Schema() == nil || len(idRef.Pointers()) == 0 || idRef.IsNil() {
		t.Fatalf("IdRef methods failed")
	}
	var idRefList chatroom.IdRefList
	idRefList = append(idRefList, idRef)
	if idRefList.Len() != 1 || idRefList.At(0) == nil || idRefList.IsNil() {
		t.Fatalf("IdRefList methods failed")
	}

	// Exercise args models
	odArgs := &chatroom.OpenDirectArgs{TenantId: "t1", OtherUserId: "u2"}
	if odArgs.ModelName() == "" || odArgs.Schema() == nil || len(odArgs.Pointers()) == 0 || odArgs.IsNil() {
		t.Fatalf("OpenDirectArgs failed")
	}
	var odList chatroom.OpenDirectArgsList
	odList = append(odList, odArgs)
	if odList.Len() != 1 || odList.At(0) == nil {
		t.Fatalf("OpenDirectArgsList failed")
	}

	lrArgs := &chatroom.ListRoomsArgs{TenantId: "t1"}
	if lrArgs.ModelName() == "" || lrArgs.Schema() == nil || len(lrArgs.Pointers()) == 0 || lrArgs.IsNil() {
		t.Fatalf("ListRoomsArgs failed")
	}
	var lrList chatroom.ListRoomsArgsList
	lrList = append(lrList, lrArgs)
	if lrList.Len() != 1 {
		t.Fatalf("ListRoomsArgsList failed")
	}

	lmArgs := &chatroom.ListMessagesArgs{TenantId: "t1", RoomId: "r1"}
	if lmArgs.ModelName() == "" || lmArgs.Schema() == nil || len(lmArgs.Pointers()) == 0 || lmArgs.IsNil() {
		t.Fatalf("ListMessagesArgs failed")
	}
	var lmList chatroom.ListMessagesArgsList
	lmList = append(lmList, lmArgs)
	if lmList.Len() != 1 {
		t.Fatalf("ListMessagesArgsList failed")
	}

	smArgs := &chatroom.SendMessageArgs{TenantId: "t1", RoomId: "r1", Body: "test"}
	if smArgs.ModelName() == "" || smArgs.Schema() == nil || len(smArgs.Pointers()) == 0 || smArgs.IsNil() {
		t.Fatalf("SendMessageArgs failed")
	}
	var smList chatroom.SendMessageArgsList
	smList = append(smList, smArgs)
	if smList.Len() != 1 {
		t.Fatalf("SendMessageArgsList failed")
	}

	mrArgs := &chatroom.MarkReadArgs{TenantId: "t1", RoomId: "r1"}
	if mrArgs.ModelName() == "" || mrArgs.Schema() == nil || len(mrArgs.Pointers()) == 0 || mrArgs.IsNil() {
		t.Fatalf("MarkReadArgs failed")
	}
	var mrList chatroom.MarkReadArgsList
	mrList = append(mrList, mrArgs)
	if mrList.Len() != 1 {
		t.Fatalf("MarkReadArgsList failed")
	}

	hbArgs := &chatroom.HeartbeatArgs{TenantId: "t1"}
	if hbArgs.ModelName() == "" || hbArgs.Schema() == nil || len(hbArgs.Pointers()) == 0 || hbArgs.IsNil() {
		t.Fatalf("HeartbeatArgs failed")
	}
	var hbList chatroom.HeartbeatArgsList
	hbList = append(hbList, hbArgs)
	if hbList.Len() != 1 {
		t.Fatalf("HeartbeatArgsList failed")
	}

	lgArgs := &chatroom.ListGroupsArgs{TenantId: "t1"}
	if lgArgs.ModelName() == "" || lgArgs.Schema() == nil || len(lgArgs.Pointers()) == 0 || lgArgs.IsNil() {
		t.Fatalf("ListGroupsArgs failed")
	}
	var lgList chatroom.ListGroupsArgsList
	lgList = append(lgList, lgArgs)
	if lgList.Len() != 1 {
		t.Fatalf("ListGroupsArgsList failed")
	}

	sgArgs := &chatroom.SaveGroupArgs{TenantId: "t1", Name: "g"}
	if sgArgs.ModelName() == "" || sgArgs.Schema() == nil || len(sgArgs.Pointers()) == 0 || sgArgs.IsNil() {
		t.Fatalf("SaveGroupArgs failed")
	}
	var sgList chatroom.SaveGroupArgsList
	sgList = append(sgList, sgArgs)
	if sgList.Len() != 1 {
		t.Fatalf("SaveGroupArgsList failed")
	}

	sgmArgs := &chatroom.SetGroupMembersArgs{TenantId: "t1", RoomId: "r1"}
	if sgmArgs.ModelName() == "" || sgmArgs.Schema() == nil || len(sgmArgs.Pointers()) == 0 || sgmArgs.IsNil() {
		t.Fatalf("SetGroupMembersArgs failed")
	}
	var sgmList chatroom.SetGroupMembersArgsList
	sgmList = append(sgmList, sgmArgs)
	if sgmList.Len() != 1 {
		t.Fatalf("SetGroupMembersArgsList failed")
	}

	lgmArgs := &chatroom.ListGroupMembersArgs{TenantId: "t1", RoomId: "r1"}
	if lgmArgs.ModelName() == "" || lgmArgs.Schema() == nil || len(lgmArgs.Pointers()) == 0 || lgmArgs.IsNil() {
		t.Fatalf("ListGroupMembersArgs failed")
	}
	var lgmList chatroom.ListGroupMembersArgsList
	lgmList = append(lgmList, lgmArgs)
	if lgmList.Len() != 1 {
		t.Fatalf("ListGroupMembersArgsList failed")
	}

	dgArgs := &chatroom.DeleteGroupArgs{TenantId: "t1", Id: "r1"}
	if dgArgs.ModelName() == "" || dgArgs.Schema() == nil || len(dgArgs.Pointers()) == 0 || dgArgs.IsNil() {
		t.Fatalf("DeleteGroupArgs failed")
	}
	var dgList chatroom.DeleteGroupArgsList
	dgList = append(dgList, dgArgs)
	if dgList.Len() != 1 {
		t.Fatalf("DeleteGroupArgsList failed")
	}

	part := &chatroom.Participant{UserId: "u1", Label: "Alice", Online: true}
	if part.ModelName() == "" || part.Schema() == nil || len(part.Pointers()) == 0 || part.IsNil() {
		t.Fatalf("Participant failed")
	}
	var partList chatroom.ParticipantList
	partList = append(partList, part)
	if partList.Len() != 1 {
		t.Fatalf("ParticipantList failed")
	}

	rs := &chatroom.RoomSummary{RoomId: "r1", Kind: "direct", Title: "Bob"}
	if rs.ModelName() == "" || rs.Schema() == nil || len(rs.Pointers()) == 0 || rs.IsNil() {
		t.Fatalf("RoomSummary failed")
	}
	var rsList chatroom.RoomSummaryList
	rsList = append(rsList, rs)
	if rsList.Len() != 1 {
		t.Fatalf("RoomSummaryList failed")
	}

	mv := &chatroom.MessageView{Id: "m1", RoomId: "r1", Body: "hi"}
	if mv.ModelName() == "" || mv.Schema() == nil || len(mv.Pointers()) == 0 || mv.IsNil() {
		t.Fatalf("MessageView failed")
	}
	var mvList chatroom.MessageViewList
	mvList = append(mvList, mv)
	if mvList.Len() != 1 {
		t.Fatalf("MessageViewList failed")
	}

	mse := &chatroom.MessageSentEvent{TenantId: "t1", RoomId: "r1"}
	if mse.ModelName() == "" || mse.Schema() == nil || len(mse.Pointers()) == 0 || mse.IsNil() {
		t.Fatalf("MessageSentEvent failed")
	}
	var mseList chatroom.MessageSentEventList
	mseList = append(mseList, mse)
	if mseList.Len() != 1 {
		t.Fatalf("MessageSentEventList failed")
	}
}
