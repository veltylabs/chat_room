package chatroom

import (
	"webtyp.com/events"
	"webtyp.com/model"
	"webtyp.com/orm"
)

func trimSpaces(s string) string {
	runes := []rune(s)
	start := 0
	end := len(runes)

	for start < end {
		r := runes[start]
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			start++
		} else {
			break
		}
	}

	for end > start {
		r := runes[end-1]
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			end--
		} else {
			break
		}
	}

	return string(runes[start:end])
}

func (m *Module) ListMessages(tenantID, userID, roomID, afterID string, limit int) ([]MessageView, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	if tenantID == "" {
		tenantID = m.tenantID
	}
	if limit <= 0 {
		limit = DefaultMessagesPageLimit
	}

	if err := m.ensureBroadcast(tenantID); err != nil {
		return nil, err
	}

	var room Room
	err := m.db.Query(&room).
		Where("tenant_id").Eq(tenantID).
		Where("id").Eq(roomID).
		ReadOne()
	if orm.IsNotFound(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	isMem, err := m.isMember(tenantID, userID, room)
	if err != nil {
		return nil, err
	}
	if !isMem {
		return nil, ErrNotMember
	}

	var afterMsg Message
	if afterID != "" {
		err := m.db.Query(&afterMsg).
			Where("tenant_id").Eq(tenantID).
			Where("room_id").Eq(roomID).
			Where("id").Eq(afterID).
			ReadOne()
		if orm.IsNotFound(err) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
	}

	var allMsgs []Message
	qb := m.db.Query(&Message{}).
		Where("tenant_id").Eq(tenantID).
		Where("room_id").Eq(roomID)

	err = qb.OrderBy("created_at").Asc().
		ReadAll(func() model.Model { return &Message{} }, func(m model.Model) {
			allMsgs = append(allMsgs, *m.(*Message))
		})
	if err != nil {
		return nil, err
	}

	var filtered []Message
	if afterID != "" {
		for _, msg := range allMsgs {
			if msg.CreatedAt > afterMsg.CreatedAt || (msg.CreatedAt == afterMsg.CreatedAt && msg.Id > afterMsg.Id) {
				filtered = append(filtered, msg)
			}
		}
	} else {
		filtered = allMsgs
	}

	if limit > 0 && len(filtered) > limit {
		if afterID != "" {
			filtered = filtered[:limit]
		} else {
			filtered = filtered[len(filtered)-limit:]
		}
	}

	var otherReadAt int64
	if room.Kind == KindDirect {
		var otherMem Member
		err := m.db.Query(&otherMem).
			Where("tenant_id").Eq(tenantID).
			Where("room_id").Eq(roomID).
			Where("user_id").Neq(userID).
			ReadOne()
		if err == nil {
			otherReadAt = otherMem.LastReadAt
		}
	}

	var views []MessageView
	for _, msg := range filtered {
		isMine := msg.SenderId == userID
		isRead := false
		if room.Kind == KindDirect && isMine {
			isRead = otherReadAt >= msg.CreatedAt
		}

		views = append(views, MessageView{
			Id:          msg.Id,
			RoomId:      msg.RoomId,
			SenderId:    msg.SenderId,
			SenderLabel: msg.SenderLabel,
			Body:        msg.Body,
			CreatedAt:   msg.CreatedAt,
			Mine:        isMine,
			Read:        isRead,
		})
	}

	return views, nil
}

func (m *Module) SendMessage(tenantID, userID, roomID, body string) (MessageView, error) {
	if userID == "" {
		return MessageView{}, ErrUnauthenticated
	}
	if tenantID == "" {
		tenantID = m.tenantID
	}

	if err := m.ensureBroadcast(tenantID); err != nil {
		return MessageView{}, err
	}

	var room Room
	err := m.db.Query(&room).
		Where("tenant_id").Eq(tenantID).
		Where("id").Eq(roomID).
		ReadOne()
	if orm.IsNotFound(err) {
		return MessageView{}, ErrNotFound
	}
	if err != nil {
		return MessageView{}, err
	}

	isMem, err := m.isMember(tenantID, userID, room)
	if err != nil {
		return MessageView{}, err
	}
	if !isMem {
		return MessageView{}, ErrNotMember
	}

	trimmedBody := trimSpaces(body)
	if len([]rune(trimmedBody)) == 0 {
		return MessageView{}, ErrEmptyBody
	}
	if len([]rune(trimmedBody)) > MaxBodyLength {
		return MessageView{}, ErrBodyTooLong
	}

	opts, err := m.participants.ParticipantOptions(tenantID)
	if err != nil {
		return MessageView{}, err
	}

	var senderLabel string
	for _, opt := range opts {
		if opt.Key == userID {
			senderLabel = opt.Value
			break
		}
	}
	if senderLabel == "" {
		senderLabel = userID
	}

	now := m.nowNano()
	msgID := m.ids.NewID()
	msg := Message{
		Id:          msgID,
		TenantId:    tenantID,
		RoomId:      roomID,
		SenderId:    userID,
		SenderLabel: senderLabel,
		Body:        trimmedBody,
		CreatedAt:   now,
	}

	err = m.db.Create(&msg)
	if err != nil {
		return MessageView{}, err
	}

	_ = m.MarkRead(tenantID, userID, roomID)

	recipients, err := m.getRoomRecipientIDs(tenantID, room, userID)
	if err == nil && m.publisher != nil {
		m.publisher.Publish(events.Event{
			Topic: TopicMessageSent,
			Payload: &MessageSentEvent{
				TenantId:  tenantID,
				RoomId:    roomID,
				MessageId: msgID,
				SenderId:  userID,
			},
		})

		for _, rec := range recipients {
			m.publisher.Publish(events.Event{
				Topic: InboxTopic(rec),
			})
		}
	}

	var otherReadAt int64
	if room.Kind == KindDirect {
		var otherMem Member
		err := m.db.Query(&otherMem).
			Where("tenant_id").Eq(tenantID).
			Where("room_id").Eq(roomID).
			Where("user_id").Neq(userID).
			ReadOne()
		if err == nil {
			otherReadAt = otherMem.LastReadAt
		}
	}

	return MessageView{
		Id:          msg.Id,
		RoomId:      msg.RoomId,
		SenderId:    msg.SenderId,
		SenderLabel: msg.SenderLabel,
		Body:        msg.Body,
		CreatedAt:   msg.CreatedAt,
		Mine:        true,
		Read:        room.Kind == KindDirect && otherReadAt >= msg.CreatedAt,
	}, nil
}

func (m *Module) MarkRead(tenantID, userID, roomID string) error {
	if userID == "" {
		return ErrUnauthenticated
	}
	if tenantID == "" {
		tenantID = m.tenantID
	}

	if err := m.ensureBroadcast(tenantID); err != nil {
		return err
	}

	var room Room
	err := m.db.Query(&room).
		Where("tenant_id").Eq(tenantID).
		Where("id").Eq(roomID).
		ReadOne()
	if orm.IsNotFound(err) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	isMem, err := m.isMember(tenantID, userID, room)
	if err != nil {
		return err
	}
	if !isMem {
		return ErrNotMember
	}

	now := m.nowNano()

	var mem Member
	err = m.db.Query(&mem).
		Where("tenant_id").Eq(tenantID).
		Where("room_id").Eq(roomID).
		Where("user_id").Eq(userID).
		ReadOne()

	if err == nil {
		mem.LastReadAt = now
		return m.db.UpdateFields(&mem, []string{"last_read_at"},
			orm.Eq(Member_.TenantId, tenantID),
			orm.Eq(Member_.RoomId, roomID),
			orm.Eq(Member_.UserId, userID),
		)
	}

	if orm.IsNotFound(err) {
		newMem := Member{
			TenantId:   tenantID,
			RoomId:     roomID,
			UserId:     userID,
			LastReadAt: now,
		}
		return m.db.Create(&newMem)
	}

	return err
}

func (m *Module) getRoomRecipientIDs(tenantID string, room Room, senderID string) ([]string, error) {
	if room.Kind == KindBroadcast {
		opts, err := m.participants.ParticipantOptions(tenantID)
		if err != nil {
			return nil, err
		}
		var recs []string
		for _, opt := range opts {
			if opt.Key != senderID {
				recs = append(recs, opt.Key)
			}
		}
		return recs, nil
	}

	var members []Member
	err := m.db.Query(&Member{}).
		Where("tenant_id").Eq(tenantID).
		Where("room_id").Eq(room.Id).
		ReadAll(func() model.Model { return &Member{} }, func(m model.Model) {
			members = append(members, *m.(*Member))
		})
	if err != nil {
		return nil, err
	}

	var recs []string
	for _, mem := range members {
		if mem.UserId != senderID {
			recs = append(recs, mem.UserId)
		}
	}
	return recs, nil
}
