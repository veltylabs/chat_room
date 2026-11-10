package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
	"webtyp.com/time"
)

func (m *Module) nowNano() time.Nano {
	if m.clock != nil {
		return m.clock()
	}
	return time.NowNano()
}

func (m *Module) ListParticipants(tenantID, userID string) ([]Participant, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	if tenantID == "" {
		tenantID = m.tenantID
	}
	if tenantID != m.tenantID {
		return []Participant{}, nil
	}

	opts, err := m.participants.ParticipantOptions(tenantID)
	if err != nil {
		return nil, err
	}

	presences, err := m.getPresences()
	if err != nil {
		return nil, err
	}

	now := m.nowNano()
	var res []Participant

	for _, opt := range opts {
		if opt.Key == userID {
			continue
		}
		var lastSeen time.Nano
		for _, p := range presences {
			if p.UserId == opt.Key {
				lastSeen = time.Nano(p.LastSeen)
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
	if tenantID != m.tenantID {
		return RoomSummary{}, ErrNotFound
	}
	if otherUserID == userID {
		return RoomSummary{}, ErrSelfDirect
	}

	otherLabel, ok, err := m.isParticipant(otherUserID)
	if err != nil {
		return RoomSummary{}, err
	}
	if !ok {
		return RoomSummary{}, ErrUnknownUser
	}
	title := otherLabel
	if title == "" {
		title = otherUserID
	}

	directKey := makeDirectKey(userID, otherUserID)
	low, high := userID, otherUserID
	if low > high {
		low, high = high, low
	}

	var room Room
	err = m.store.Atomic(func(tx crud.Tx) error {
		row, found, err := firstRow(tx, func() model.Model { return &Room{} },
			storage.Eq(Room_.Kind, KindDirect),
			storage.Eq(Room_.DirectKey, directKey))
		if err != nil {
			return err
		}
		if found {
			room = *row.(*Room)
			return nil
		}
		now := m.nowNano()
		roomID := m.ids.NewID()
		room = Room{
			Id:        roomID,
			Kind:      KindDirect,
			DirectKey: directKey,
			CreatedAt: now,
		}
		if err := tx.Create(&room); err != nil {
			return err
		}
		if err := tx.Create(&Member{RoomId: roomID, UserId: userID}); err != nil {
			return err
		}
		if err := tx.Create(&Member{RoomId: roomID, UserId: otherUserID}); err != nil {
			return err
		}
		direct := Direct{
			UserA:     low,
			UserB:     high,
			RoomId:    roomID,
			CreatedAt: now,
		}
		return tx.Create(&direct)
	})
	if err != nil {
		return RoomSummary{}, err
	}

	return m.getRoomSummary(room, userID, title)
}

func (m *Module) ListRooms(tenantID, userID string) ([]RoomSummary, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	if tenantID == "" {
		tenantID = m.tenantID
	}
	if tenantID != m.tenantID {
		return []RoomSummary{}, nil
	}

	if err := m.ensureBroadcast(); err != nil {
		return nil, err
	}

	opts, err := m.participants.ParticipantOptions(tenantID)
	if err != nil {
		return nil, err
	}

	userMemberRows, err := allRows(m.store, func() model.Model { return &Member{} }, nil, storage.Eq(Member_.UserId, userID))
	if err != nil {
		return nil, err
	}

	var memberRoomIDs []string
	for _, row := range userMemberRows {
		memberRoomIDs = append(memberRoomIDs, row.(*Member).RoomId)
	}

	broadcastRows, err := allRows(m.store, func() model.Model { return &Room{} }, nil, storage.Eq(Room_.Kind, KindBroadcast))
	if err != nil {
		return nil, err
	}

	var allRooms []Room
	for _, row := range broadcastRows {
		allRooms = append(allRooms, *row.(*Room))
	}

	if len(memberRoomIDs) > 0 {
		memRows, err := allRows(m.store, func() model.Model { return &Room{} }, nil, storage.In(Room_.Id, memberRoomIDs))
		if err != nil {
			return nil, err
		}
		for _, row := range memRows {
			r := *row.(*Room)
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

		s, err := m.getRoomSummary(room, userID, title)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}

	return summaries, nil
}

// ensureBroadcast creates the tenant's general room the first time anyone needs it.
func (m *Module) ensureBroadcast() error {
	if _, found, err := firstRow(m.store, func() model.Model { return &Room{} }, storage.Eq(Room_.Kind, KindBroadcast)); err != nil || found {
		return err
	}
	return m.store.Atomic(func(tx crud.Tx) error {
		if _, found, err := firstRow(tx, func() model.Model { return &Room{} }, storage.Eq(Room_.Kind, KindBroadcast)); err != nil || found {
			return err
		}
		return tx.Create(&Room{Id: m.ids.NewID(), Kind: KindBroadcast, Name: BroadcastName, CreatedAt: m.nowNano()})
	})
}

func (m *Module) getRoomSummary(room Room, userID string, title string) (RoomSummary, error) {
	msgRows, err := allRows(m.store, func() model.Model { return &Message{} },
		[]storage.Order{storage.Asc(Message_.CreatedAt)},
		storage.Eq(Message_.RoomId, room.Id))
	if err != nil {
		return RoomSummary{}, err
	}

	var lastMsgAt time.Nano
	var lastPreview string
	if len(msgRows) > 0 {
		lastMsg := msgRows[len(msgRows)-1].(*Message)
		lastMsgAt = lastMsg.CreatedAt
		lastPreview = truncateString(lastMsg.Body, PreviewLength)
	}

	var lastReadAt time.Nano
	row, found, err := firstRow(m.store, func() model.Model { return &Member{} },
		storage.Eq(Member_.RoomId, room.Id),
		storage.Eq(Member_.UserId, userID))
	if err != nil {
		return RoomSummary{}, err
	}
	if found {
		lastReadAt = row.(*Member).LastReadAt
	}

	var unread int64
	for _, r := range msgRows {
		msg := r.(*Message)
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
	var idx = -1
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
