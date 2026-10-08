package chatroom

import (
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/time"
)

func (m *Module) nowNano() int64 {
	if m.clock != nil {
		return m.clock()
	}
	return time.Now()
}

func (m *Module) ListParticipants(tenantID, userID string) ([]Participant, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	if tenantID == "" {
		tenantID = m.tenantID
	}

	opts, err := m.participants.ParticipantOptions(tenantID)
	if err != nil {
		return nil, err
	}

	presences, err := m.getPresences(tenantID)
	if err != nil {
		return nil, err
	}

	now := m.nowNano()
	var res []Participant

	for _, opt := range opts {
		if opt.Key == userID {
			continue
		}
		var lastSeen int64
		for _, p := range presences {
			if p.UserId == opt.Key {
				lastSeen = p.LastSeen
				break
			}
		}
		res = append(res, Participant{
			UserId: opt.Key,
			Label:  opt.Value,
			Online: isOnline(now, lastSeen),
		})
	}

	return res, nil
}

func (m *Module) OpenDirect(tenantID, userID, otherUserID string) (RoomSummary, error) {
	if userID == "" {
		return RoomSummary{}, ErrUnauthenticated
	}
	if tenantID == "" {
		tenantID = m.tenantID
	}
	if otherUserID == userID {
		return RoomSummary{}, ErrSelfDirect
	}

	opts, err := m.participants.ParticipantOptions(tenantID)
	if err != nil {
		return RoomSummary{}, err
	}

	var otherFound bool
	var otherLabel string
	for _, opt := range opts {
		if opt.Key == otherUserID {
			otherFound = true
			otherLabel = opt.Value
			break
		}
	}
	if !otherFound {
		return RoomSummary{}, ErrUnknownUser
	}

	directKey := makeDirectKey(userID, otherUserID)

	var room Room
	err = m.db.Query(&room).
		Where("tenant_id").Eq(tenantID).
		Where("kind").Eq(KindDirect).
		Where("direct_key").Eq(directKey).
		ReadOne()

	if err != nil && !orm.IsNotFound(err) {
		return RoomSummary{}, err
	}

	if orm.IsNotFound(err) {
		now := m.nowNano()
		roomID := m.ids.NewID()
		room = Room{
			Id:        roomID,
			TenantId:  tenantID,
			Kind:      KindDirect,
			DirectKey: directKey,
			CreatedAt: now,
		}

		m1 := Member{
			TenantId: tenantID,
			RoomId:   roomID,
			UserId:   userID,
		}
		m2 := Member{
			TenantId: tenantID,
			RoomId:   roomID,
			UserId:   otherUserID,
		}

		err = m.db.Tx(func(tx *orm.DB) error {
			if err := tx.Create(&room); err != nil {
				return err
			}
			if err := tx.Create(&m1); err != nil {
				return err
			}
			return tx.Create(&m2)
		})
		if err != nil {
			return RoomSummary{}, err
		}
	}

	return m.getRoomSummary(tenantID, userID, room, otherLabel)
}

func (m *Module) ListRooms(tenantID, userID string) ([]RoomSummary, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	if tenantID == "" {
		tenantID = m.tenantID
	}

	if err := m.ensureBroadcast(tenantID); err != nil {
		return nil, err
	}

	opts, err := m.participants.ParticipantOptions(tenantID)
	if err != nil {
		return nil, err
	}

	var userMembers []Member
	err = m.db.Query(&Member{}).
		Where("tenant_id").Eq(tenantID).
		Where("user_id").Eq(userID).
		ReadAll(func() model.Model { return &Member{} }, func(m model.Model) {
			userMembers = append(userMembers, *m.(*Member))
		})
	if err != nil {
		return nil, err
	}

	var memberRoomIDs []string
	for _, mem := range userMembers {
		memberRoomIDs = append(memberRoomIDs, mem.RoomId)
	}

	var broadcastRooms []Room
	err = m.db.Query(&Room{}).
		Where("tenant_id").Eq(tenantID).
		Where("kind").Eq(KindBroadcast).
		ReadAll(func() model.Model { return &Room{} }, func(m model.Model) {
			broadcastRooms = append(broadcastRooms, *m.(*Room))
		})
	if err != nil {
		return nil, err
	}

	var allRooms []Room
	for _, br := range broadcastRooms {
		allRooms = append(allRooms, br)
	}

	if len(memberRoomIDs) > 0 {
		var memRooms []Room
		err = m.db.Query(&Room{}).
			Where("tenant_id").Eq(tenantID).
			Where("id").In(memberRoomIDs).
			ReadAll(func() model.Model { return &Room{} }, func(m model.Model) {
				memRooms = append(memRooms, *m.(*Room))
			})
		if err != nil {
			return nil, err
		}
		for _, r := range memRooms {
			// Posting in General makes the sender a member of it; it is already
			// listed above as the tenant's broadcast room.
			if r.Kind == KindBroadcast {
				continue
			}
			allRooms = append(allRooms, r)
		}
	}

	var summaries []RoomSummary
	for _, room := range allRooms {
		var title string
		if room.Kind == KindDirect {
			var otherID string
			parts := splitDirectKey(room.DirectKey)
			if parts[0] == userID {
				otherID = parts[1]
			} else {
				otherID = parts[0]
			}
			for _, kv := range opts {
				if kv.Key == otherID {
					title = kv.Value
					break
				}
			}
			if title == "" {
				title = otherID
			}
		} else {
			title = room.Name
		}

		s, err := m.getRoomSummary(tenantID, userID, room, title)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}

	return summaries, nil
}

func (m *Module) ensureBroadcast(tenantID string) error {
	var found bool
	var br Room
	err := m.db.Query(&br).
		Where("tenant_id").Eq(tenantID).
		Where("kind").Eq(KindBroadcast).
		ReadOne()
	if err == nil {
		found = true
	} else if !orm.IsNotFound(err) {
		return err
	}

	if found {
		return nil
	}

	now := m.nowNano()
	room := Room{
		Id:        m.ids.NewID(),
		TenantId:  tenantID,
		Kind:      KindBroadcast,
		Name:      BroadcastName,
		CreatedAt: now,
	}
	err = m.db.Create(&room)
	if err != nil {
		return err
	}
	return nil
}

func (m *Module) isMember(tenantID, userID string, room Room) (bool, error) {
	if room.Kind == KindBroadcast {
		opts, err := m.participants.ParticipantOptions(tenantID)
		if err != nil {
			return false, err
		}
		for _, opt := range opts {
			if opt.Key == userID {
				return true, nil
			}
		}
		return false, nil
	}

	var mem Member
	err := m.db.Query(&mem).
		Where("tenant_id").Eq(tenantID).
		Where("room_id").Eq(room.Id).
		Where("user_id").Eq(userID).
		ReadOne()
	if err == nil {
		return true, nil
	}
	if orm.IsNotFound(err) {
		return false, nil
	}
	return false, err
}

func (m *Module) getRoomSummary(tenantID, userID string, room Room, title string) (RoomSummary, error) {
	var messages []Message
	err := m.db.Query(&Message{}).
		Where("tenant_id").Eq(tenantID).
		Where("room_id").Eq(room.Id).
		OrderBy("created_at").Asc().
		ReadAll(func() model.Model { return &Message{} }, func(m model.Model) {
			messages = append(messages, *m.(*Message))
		})
	if err != nil {
		return RoomSummary{}, err
	}

	var lastMsgAt int64
	var lastPreview string
	if len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		lastMsgAt = lastMsg.CreatedAt
		lastPreview = truncateString(lastMsg.Body, PreviewLength)
	}

	var lastReadAt int64
	var mem Member
	err = m.db.Query(&mem).
		Where("tenant_id").Eq(tenantID).
		Where("room_id").Eq(room.Id).
		Where("user_id").Eq(userID).
		ReadOne()
	if err == nil {
		lastReadAt = mem.LastReadAt
	}

	var unread int64
	for _, msg := range messages {
		if msg.SenderId != userID && msg.CreatedAt > lastReadAt {
			unread++
		}
	}

	return RoomSummary{
		RoomId:        room.Id,
		Kind:          room.Kind,
		Title:         title,
		Unread:        unread,
		LastMessageAt: lastMsgAt,
		LastPreview:   lastPreview,
	}, nil
}

func makeDirectKey(a, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}

func splitDirectKey(key string) []string {
	var idx int = -1
	for i := 0; i < len(key); i++ {
		if key[i] == '|' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return []string{key, key}
	}
	return []string{key[:idx], key[idx+1:]}
}

func truncateString(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}
