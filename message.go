package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/events"
	"webtyp.com/model"
	"webtyp.com/storage"
	"webtyp.com/time"
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
	if tenantID != m.tenantID {
		return nil, ErrNotFound
	}
	if limit <= 0 {
		limit = DefaultMessagesPageLimit
	}
	if err := m.ensureBroadcast(); err != nil {
		return nil, err
	}
	room, err := roomOf(m.store, roomID)
	if err != nil {
		return nil, err
	}
	isMember, err := m.isRoomMember(m.store, userID, room)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotFound // a person outside the room must not learn it exists
	}
	conditions := []storage.Condition{storage.Eq(Message_.RoomId, roomID)}
	order := []storage.Order{storage.Desc(Message_.CreatedAt)}
	var after Message
	if afterID != "" {
		if err := m.store.ReadOne(&after, afterID); err != nil || after.RoomId != roomID {
			return nil, ErrNotFound
		}
		conditions = append(conditions, storage.Gte(Message_.CreatedAt, int64(after.CreatedAt)))
		order = []storage.Order{storage.Asc(Message_.CreatedAt)}
	}
	var messages []Message
	err = m.store.ReadAll(func() model.Model { return &Message{} },
		crud.Query{Conditions: conditions, Order: order, Limit: int64(limit) + 1},
		func(r model.Model) { messages = append(messages, *r.(*Message)) })
	if err != nil {
		return nil, err
	}
	if afterID != "" {
		var newer []Message
		for _, message := range messages {
			if message.CreatedAt > after.CreatedAt || (message.CreatedAt == after.CreatedAt && message.Id > after.Id) {
				newer = append(newer, message)
			}
		}
		messages = newer
		if len(messages) > limit {
			messages = messages[:limit]
		}
	} else {
		if len(messages) > limit {
			messages = messages[:limit]
		}
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 { // newest-first read, oldest-first answer
			messages[i], messages[j] = messages[j], messages[i]
		}
	}
	var otherReadAt time.Nano
	if room.Kind == KindDirect {
		row, found, err := firstRow(m.store, func() model.Model { return &Member{} },
			storage.Eq(Member_.RoomId, roomID), storage.Neq(Member_.UserId, userID))
		if err != nil {
			return nil, err
		}
		if found {
			otherReadAt = row.(*Member).LastReadAt
		}
	}
	views := make([]MessageView, 0, len(messages))
	for _, message := range messages {
		mine := message.SenderId == userID
		views = append(views, MessageView{
			Id: message.Id, RoomId: message.RoomId, SenderId: message.SenderId, SenderLabel: message.SenderLabel,
			Body: message.Body, CreatedAt: message.CreatedAt, Mine: mine,
			IsRead: room.Kind == KindDirect && mine && otherReadAt >= message.CreatedAt,
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
	if tenantID != m.tenantID {
		return MessageView{}, ErrNotFound
	}
	if err := m.ensureBroadcast(); err != nil {
		return MessageView{}, err
	}
	message := Message{RoomId: roomID, Body: body}
	var room Room
	err := m.store.Atomic(func(tx crud.Tx) error {
		var checkErr error
		room, checkErr = m.checkSendMessage(tx, userID, &message)
		if checkErr != nil {
			return checkErr
		}
		message.Id = m.ids.NewID()
		return tx.Create(&message)
	})
	if err != nil {
		return MessageView{}, err
	}
	_ = m.MarkRead(tenantID, userID, roomID)
	recipients, recipientsErr := m.recipientIDs(room, userID)
	if recipientsErr == nil && m.publisher != nil {
		m.publisher.Publish(events.Event{Topic: TopicMessageSent, Payload: &MessageSentEvent{TenantId: tenantID, RoomId: roomID, MessageId: message.Id, SenderId: userID}})
		for _, recipient := range recipients {
			m.publisher.Publish(events.Event{Topic: InboxTopic(recipient)})
		}
	}
	var otherReadAt time.Nano
	if room.Kind == KindDirect {
		row, found, err := firstRow(m.store, func() model.Model { return &Member{} },
			storage.Eq(Member_.RoomId, roomID), storage.Neq(Member_.UserId, userID))
		if err == nil && found {
			otherReadAt = row.(*Member).LastReadAt
		}
	}
	return MessageView{Id: message.Id, RoomId: message.RoomId, SenderId: message.SenderId, SenderLabel: message.SenderLabel,
		Body: message.Body, CreatedAt: message.CreatedAt, Mine: true,
		IsRead: room.Kind == KindDirect && otherReadAt >= message.CreatedAt}, nil
}

func (m *Module) MarkRead(tenantID, userID, roomID string) error {
	if userID == "" {
		return ErrUnauthenticated
	}
	if tenantID == "" {
		tenantID = m.tenantID
	}
	if tenantID != m.tenantID {
		return ErrNotFound
	}
	if err := m.ensureBroadcast(); err != nil {
		return err
	}
	return m.store.Atomic(func(tx crud.Tx) error {
		room, err := roomOf(tx, roomID)
		if err != nil {
			return err
		}
		isMember, err := m.isRoomMember(tx, userID, room)
		if err != nil {
			return err
		}
		if !isMember {
			return ErrNotMember
		}
		now := m.nowNano()
		row, found, err := firstRow(tx, func() model.Model { return &Member{} },
			storage.Eq(Member_.RoomId, roomID), storage.Eq(Member_.UserId, userID))
		if err != nil {
			return err
		}
		if found {
			member := row.(*Member)
			member.LastReadAt = now
			return tx.UpdateFields(member, model.JoinKey(member.RoomId, member.UserId), []string{Member_.LastReadAt})
		}
		return tx.Create(&Member{RoomId: roomID, UserId: userID, LastReadAt: now})
	})
}

func (m *Module) recipientIDs(room Room, senderID string) ([]string, error) {
	if room.Kind == KindBroadcast {
		opts, err := m.participants.ParticipantOptions(m.tenantID)
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

	rows, err := allRows(m.store, func() model.Model { return &Member{} }, nil, storage.Eq(Member_.RoomId, room.Id))
	if err != nil {
		return nil, err
	}
	var recs []string
	for _, row := range rows {
		mem := row.(*Member)
		if mem.UserId != senderID {
			recs = append(recs, mem.UserId)
		}
	}
	return recs, nil
}
