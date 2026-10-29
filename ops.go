package chatroom

import (
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/rpc"
	"webtyp.com/view"
)

const (
	OpListParticipants    = "list_participants"
	OpHeartbeat           = "heartbeat"
	OpOpenDirect          = "open_direct"
	OpListRooms           = "list_rooms"
	OpListMessages        = "list_messages"
	OpSendMessage         = "send_message"
	OpMarkRead            = "mark_read"
	OpListGroups          = "list_groups"
	OpSaveGroup           = "save_group"
	OpSetGroupMembers     = "set_group_members"
	OpListGroupMembers    = "list_group_members"
	OpListGroupCandidates = "list_group_candidates"
	OpDeleteGroup         = "delete_group"
)

var _ router.OperationModule = (*Module)(nil)

func (m *Module) MountOperations(reg router.OperationRegistry) {
	reg.Operation(OpListParticipants, m.handleListParticipants).
		Authenticated().
		Accepts(&ListRoomsArgs{})

	reg.Operation(OpHeartbeat, m.handleHeartbeat).
		Authenticated().
		Accepts(&HeartbeatArgs{})

	reg.Operation(OpOpenDirect, m.handleOpenDirect).
		Authenticated().
		Accepts(&OpenDirectArgs{})

	reg.Operation(OpListRooms, m.handleListRooms).
		Authenticated().
		Accepts(&ListRoomsArgs{})

	reg.Operation(OpListMessages, m.handleListMessages).
		Authenticated().
		Accepts(&ListMessagesArgs{})

	reg.Operation(OpSendMessage, m.handleSendMessage).
		Authenticated().
		Accepts(&SendMessageArgs{})

	reg.Operation(OpMarkRead, m.handleMarkRead).
		Authenticated().
		Accepts(&MarkReadArgs{})

	reg.Operation(OpListGroups, m.handleListGroups).
		Requires(model.ResourceOf(&RoomModel), model.Read).
		Accepts(&ListGroupsArgs{})

	view.MountSave(reg, OpSaveGroup, func(r *Room) error { out, err := m.SaveGroup(r.TenantId, r.Id, r.Name); *r = out; return err }).Requires(model.ResourceOf(&RoomModel), model.Create|model.Update)

	reg.Operation(OpSetGroupMembers, m.handleSetGroupMembers).
		Requires(model.ResourceOf(&RoomModel), model.Update).
		Accepts(&SetGroupMembersArgs{})

	reg.Operation(OpListGroupMembers, m.handleListGroupMembers).
		Requires(model.ResourceOf(&RoomModel), model.Read).
		Accepts(&ListGroupMembersArgs{})

	reg.Operation(OpListGroupCandidates, m.handleListGroupCandidates).
		Requires(model.ResourceOf(&RoomModel), model.Read).
		Accepts(&ListRoomsArgs{})

	view.MountDelete(reg, OpDeleteGroup, func(ids []string) error {
		for _, id := range ids {
			if err := m.DeleteGroup(m.tenantID, id); err != nil {
				return err
			}
		}
		return nil
	}).Requires(model.ResourceOf(&RoomModel), model.Delete)
}

func (m *Module) handleListParticipants(ctx router.Context) {
	var args ListRoomsArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	userID := ctx.UserID()
	participants, err := m.ListParticipants(args.TenantId, userID)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	var list ParticipantList
	for i := range participants {
		list = append(list, &participants[i])
	}
	_ = ctx.Encode(&list)
}

func (m *Module) handleHeartbeat(ctx router.Context) {
	var args HeartbeatArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	userID := ctx.UserID()
	participants, err := m.Heartbeat(args.TenantId, userID)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	var list ParticipantList
	for i := range participants {
		list = append(list, &participants[i])
	}
	_ = ctx.Encode(&list)
}

func (m *Module) handleOpenDirect(ctx router.Context) {
	var args OpenDirectArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	userID := ctx.UserID()
	summary, err := m.OpenDirect(args.TenantId, userID, args.OtherUserId)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	_ = ctx.Encode(&summary)
}

func (m *Module) handleListRooms(ctx router.Context) {
	var args ListRoomsArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	userID := ctx.UserID()
	summaries, err := m.ListRooms(args.TenantId, userID)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	var list RoomSummaryList
	for i := range summaries {
		list = append(list, &summaries[i])
	}
	_ = ctx.Encode(&list)
}

func (m *Module) handleListMessages(ctx router.Context) {
	var args ListMessagesArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	userID := ctx.UserID()
	messages, err := m.ListMessages(args.TenantId, userID, args.RoomId, args.AfterId, int(args.Limit))
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	var list MessageViewList
	for i := range messages {
		list = append(list, &messages[i])
	}
	_ = ctx.Encode(&list)
}

func (m *Module) handleSendMessage(ctx router.Context) {
	var args SendMessageArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	if err := args.Validate(model.ActionCreate); err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	userID := ctx.UserID()
	msgView, err := m.SendMessage(args.TenantId, userID, args.RoomId, args.Body)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	_ = ctx.Encode(&msgView)
}

func (m *Module) handleMarkRead(ctx router.Context) {
	var args MarkReadArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	if err := args.Validate(model.ActionUpdate); err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	userID := ctx.UserID()
	err := m.MarkRead(args.TenantId, userID, args.RoomId)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	ctx.WriteStatus(200)
}

func (m *Module) handleListGroups(ctx router.Context) {
	var args ListGroupsArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	groups, err := m.ListGroups(args.TenantId)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	var list RoomList
	for i := range groups {
		list = append(list, &groups[i])
	}
	_ = ctx.Encode(&list)
}

func (m *Module) handleSetGroupMembers(ctx router.Context) {
	var args SetGroupMembersArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	if err := args.Validate(model.ActionUpdate); err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	userIDs := make([]string, len(args.UserIds))
	for i, ref := range args.UserIds {
		userIDs[i] = ref.Id
	}

	err := m.SetGroupMembers(args.TenantId, args.RoomId, userIDs)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	ctx.WriteStatus(200)
}

func (m *Module) handleListGroupMembers(ctx router.Context) {
	var args ListGroupMembersArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	participants, err := m.ListGroupMembers(args.TenantId, args.RoomId)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	var list ParticipantList
	for i := range participants {
		list = append(list, &participants[i])
	}
	_ = ctx.Encode(&list)
}

func (m *Module) handleListGroupCandidates(ctx router.Context) {
	var args ListRoomsArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	participants, err := m.ListGroupCandidates(args.TenantId)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}

	var list ParticipantList
	for i := range participants {
		list = append(list, &participants[i])
	}
	_ = ctx.Encode(&list)
}
