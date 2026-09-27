package chatroom

import (
	"webtyp.com/model"
	"webtyp.com/router"
)

const (
	OpListParticipants = "list_participants"
	OpHeartbeat        = "heartbeat"
	OpOpenDirect        = "open_direct"
	OpListRooms        = "list_rooms"
	OpListMessages     = "list_messages"
	OpSendMessage      = "send_message"
	OpMarkRead         = "mark_read"
	OpListGroups       = "list_groups"
	OpSaveGroup        = "save_group"
	OpSetGroupMembers  = "set_group_members"
	OpListGroupMembers = "list_group_members"
	OpDeleteGroup      = "delete_group"
)

var _ router.OperationModule = (*Module)(nil)

func (m *Module) MountOperations(reg router.OperationRegistry) {
	m.MountOps(reg)
}

func (m *Module) MountOps(reg router.OperationRegistry) {
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
		Requires(ResourceGroup, model.Read).
		Accepts(&ListGroupsArgs{})

	reg.Operation(OpSaveGroup, m.handleSaveGroup).
		Requires(ResourceGroup, model.Create|model.Update).
		Accepts(&SaveGroupArgs{})

	reg.Operation(OpSetGroupMembers, m.handleSetGroupMembers).
		Requires(ResourceGroup, model.Update).
		Accepts(&SetGroupMembersArgs{})

	reg.Operation(OpListGroupMembers, m.handleListGroupMembers).
		Requires(ResourceGroup, model.Read).
		Accepts(&ListGroupMembersArgs{})

	reg.Operation(OpDeleteGroup, m.handleDeleteGroup).
		Requires(ResourceGroup, model.Delete).
		Accepts(&DeleteGroupArgs{})
}

func writeError(ctx router.Context, err error) {
	switch err {
	case ErrUnauthenticated:
		ctx.WriteStatus(401)
	case ErrNotMember:
		ctx.WriteStatus(403)
	case ErrNotFound:
		ctx.WriteStatus(404)
	case ErrUnknownUser, ErrSelfDirect, ErrEmptyBody, ErrBodyTooLong, ErrNotGroup:
		ctx.WriteStatus(400)
	default:
		ctx.WriteStatus(500)
	}
}

func (m *Module) handleListParticipants(ctx router.Context) {
	var args ListRoomsArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		ctx.WriteStatus(400)
		return
	}

	userID := ctx.UserID()
	participants, err := m.ListParticipants(args.TenantId, userID)
	if err != nil {
		writeError(ctx, err)
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
		ctx.WriteStatus(400)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		ctx.WriteStatus(400)
		return
	}

	userID := ctx.UserID()
	participants, err := m.Heartbeat(args.TenantId, userID)
	if err != nil {
		writeError(ctx, err)
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
		ctx.WriteStatus(400)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		ctx.WriteStatus(400)
		return
	}

	userID := ctx.UserID()
	summary, err := m.OpenDirect(args.TenantId, userID, args.OtherUserId)
	if err != nil {
		writeError(ctx, err)
		return
	}

	_ = ctx.Encode(&summary)
}

func (m *Module) handleListRooms(ctx router.Context) {
	var args ListRoomsArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		ctx.WriteStatus(400)
		return
	}

	userID := ctx.UserID()
	summaries, err := m.ListRooms(args.TenantId, userID)
	if err != nil {
		writeError(ctx, err)
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
		ctx.WriteStatus(400)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		ctx.WriteStatus(400)
		return
	}

	userID := ctx.UserID()
	messages, err := m.ListMessages(args.TenantId, userID, args.RoomId, args.AfterId, int(args.Limit))
	if err != nil {
		writeError(ctx, err)
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
		ctx.WriteStatus(400)
		return
	}
	if err := args.Validate(model.ActionCreate); err != nil {
		ctx.WriteStatus(400)
		return
	}

	userID := ctx.UserID()
	msgView, err := m.SendMessage(args.TenantId, userID, args.RoomId, args.Body)
	if err != nil {
		writeError(ctx, err)
		return
	}

	_ = ctx.Encode(&msgView)
}

func (m *Module) handleMarkRead(ctx router.Context) {
	var args MarkReadArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := args.Validate(model.ActionUpdate); err != nil {
		ctx.WriteStatus(400)
		return
	}

	userID := ctx.UserID()
	err := m.MarkRead(args.TenantId, userID, args.RoomId)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.WriteStatus(200)
}

func (m *Module) handleListGroups(ctx router.Context) {
	var args ListGroupsArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		ctx.WriteStatus(400)
		return
	}

	groups, err := m.ListGroups(args.TenantId)
	if err != nil {
		writeError(ctx, err)
		return
	}

	var list RoomList
	for i := range groups {
		list = append(list, &groups[i])
	}
	_ = ctx.Encode(&list)
}

func (m *Module) handleSaveGroup(ctx router.Context) {
	var args SaveGroupArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	action := model.ActionCreate
	if args.Id != "" {
		action = model.ActionUpdate
	}
	if err := args.Validate(action); err != nil {
		ctx.WriteStatus(400)
		return
	}

	group, err := m.SaveGroup(args.TenantId, args.Id, args.Name)
	if err != nil {
		writeError(ctx, err)
		return
	}

	_ = ctx.Encode(&group)
}

func (m *Module) handleSetGroupMembers(ctx router.Context) {
	var args SetGroupMembersArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := args.Validate(model.ActionUpdate); err != nil {
		ctx.WriteStatus(400)
		return
	}

	userIDs := make([]string, len(args.UserIds))
	for i, ref := range args.UserIds {
		userIDs[i] = ref.Id
	}

	err := m.SetGroupMembers(args.TenantId, args.RoomId, userIDs)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.WriteStatus(200)
}

func (m *Module) handleListGroupMembers(ctx router.Context) {
	var args ListGroupMembersArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := args.Validate(model.ActionRead); err != nil {
		ctx.WriteStatus(400)
		return
	}

	participants, err := m.ListGroupMembers(args.TenantId, args.RoomId)
	if err != nil {
		writeError(ctx, err)
		return
	}

	var list ParticipantList
	for i := range participants {
		list = append(list, &participants[i])
	}
	_ = ctx.Encode(&list)
}

func (m *Module) handleDeleteGroup(ctx router.Context) {
	var args DeleteGroupArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := args.Validate(model.ActionDelete); err != nil {
		ctx.WriteStatus(400)
		return
	}

	err := m.DeleteGroup(args.TenantId, args.Id)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.WriteStatus(200)
}
