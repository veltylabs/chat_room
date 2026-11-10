package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/events"
	"webtyp.com/model"
	"webtyp.com/storage"
	"webtyp.com/time"
)

func (m *Module) ListGroups(search, after string, limit int64) ([]Room, error) {
	var conditions []storage.Condition
	conditions = append(conditions, storage.Eq(Room_.Kind, KindGroup))
	if search != "" {
		conditions = append(conditions, storage.Like(Room_.Name, "%"+search+"%"))
	}
	var groups []Room
	err := m.store.ReadAll(func() model.Model { return &Room{} },
		crud.Query{Conditions: conditions, After: after, Limit: limit},
		func(r model.Model) { groups = append(groups, *r.(*Room)) })
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func (m *Module) SaveGroup(tenantID, id, name string) (Room, error) {
	if tenantID == "" {
		tenantID = m.tenantID
	}
	if tenantID != m.tenantID {
		return Room{}, ErrNotFound
	}
	name, err := validGroupName(name)
	if err != nil {
		return Room{}, err
	}
	now := m.nowNano()
	var saved Room
	err = m.store.Atomic(func(tx crud.Tx) error {
		if id != "" {
			room, err := roomOf(tx, id)
			if err != nil {
				return err
			}
			if room.Kind != KindGroup {
				return ErrNotGroup
			}
			room.Name = name
			room.UpdatedAt = now
			if err := tx.UpdateFields(&room, room.Id, []string{Room_.Name, Room_.UpdatedAt}); err != nil {
				return err
			}
			saved = room
			return nil
		}
		saved = Room{
			Id:        m.ids.NewID(),
			Kind:      KindGroup,
			Name:      name,
			CreatedAt: now,
		}
		return tx.Create(&saved)
	})
	if err != nil {
		return Room{}, err
	}
	if m.publisher != nil {
		m.publisher.Publish(events.Event{
			Topic:   TopicGroupSaved,
			Payload: &saved,
		})
	}
	return saved, nil
}

func (m *Module) SetGroupMembers(tenantID, roomID string, userIDs []string) error {
	if tenantID == "" {
		tenantID = m.tenantID
	}
	if tenantID != m.tenantID {
		return ErrNotFound
	}
	var added, removed []string
	err := m.store.Atomic(func(tx crud.Tx) error {
		room, err := roomOf(tx, roomID)
		if err != nil {
			return err
		}
		if room.Kind != KindGroup {
			return ErrNotGroup
		}
		for _, userID := range userIDs {
			if _, ok, err := m.isParticipant(userID); err != nil {
				return err
			} else if !ok {
				return ErrUnknownUser
			}
		}
		rows, err := allRows(tx, func() model.Model { return &Member{} }, nil, storage.Eq(Member_.RoomId, roomID))
		if err != nil {
			return err
		}
		for _, row := range rows {
			member := row.(*Member)
			if !containsString(userIDs, member.UserId) {
				if err := tx.Delete(row, model.JoinKey(member.RoomId, member.UserId)); err != nil {
					return err
				}
				removed = append(removed, member.UserId)
			}
		}
		for _, userID := range userIDs {
			present := false
			for _, row := range rows {
				if row.(*Member).UserId == userID {
					present = true
				}
			}
			if !present {
				if err := tx.Create(&Member{RoomId: roomID, UserId: userID}); err != nil {
					return err
				}
				added = append(added, userID)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if m.publisher != nil {
		affected := append(added, removed...)
		for _, u := range affected {
			m.publisher.Publish(events.Event{
				Topic: InboxTopic(u),
			})
		}
	}
	return nil
}

func (m *Module) ListGroupCandidates(tenantID string) ([]Participant, error) {
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

	participants := make([]Participant, len(opts))
	for i, opt := range opts {
		participants[i] = Participant{
			UserId: opt.Key,
			Label:  opt.Value,
			Online: false,
		}
	}

	return participants, nil
}

func (m *Module) ListGroupMembers(tenantID, roomID string) ([]Participant, error) {
	if tenantID == "" {
		tenantID = m.tenantID
	}
	if tenantID != m.tenantID {
		return []Participant{}, nil
	}

	room, err := roomOf(m.store, roomID)
	if err != nil {
		return nil, err
	}
	if room.Kind != KindGroup {
		return nil, ErrNotGroup
	}

	memberRows, err := allRows(m.store, func() model.Model { return &Member{} }, nil, storage.Eq(Member_.RoomId, roomID))
	if err != nil {
		return nil, err
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
	var participants []Participant

	for _, row := range memberRows {
		mem := row.(*Member)
		var label string
		for _, opt := range opts {
			if opt.Key == mem.UserId {
				label = opt.Value
				break
			}
		}
		if label == "" {
			label = mem.UserId
		}

		var lastSeen time.Nano
		for _, p := range presences {
			if p.UserId == mem.UserId {
				lastSeen = time.Nano(p.LastSeen)
				break
			}
		}

		participants = append(participants, Participant{
			UserId: mem.UserId,
			Label:  label,
			Online: isOnline(now, lastSeen),
		})
	}

	return participants, nil
}

func (m *Module) DeleteGroup(tenantID, id string) error {
	if tenantID == "" {
		tenantID = m.tenantID
	}
	if tenantID != m.tenantID {
		return ErrNotFound
	}
	return m.store.Atomic(func(tx crud.Tx) error {
		room, err := roomOf(tx, id)
		if err != nil {
			return err
		}
		if room.Kind != KindGroup {
			return ErrNotGroup
		}
		if err := deleteGroupRows(tx, id); err != nil {
			return err
		}
		return tx.Delete(&room, id)
	})
}

func (m *Module) PurgeExpired(tenantID string) (int, error) {
	if tenantID == "" {
		tenantID = m.tenantID
	}
	if tenantID != m.tenantID {
		return 0, nil
	}

	now := m.nowNano()
	cutoff := now - (time.Unix(m.retentionDays) * time.Day).Nano()

	count := 0
	for {
		rows, err := allRows(m.store, func() model.Model { return &Message{} }, nil, storage.Lt(Message_.CreatedAt, int64(cutoff)))
		if err != nil {
			return count, err
		}
		if len(rows) == 0 {
			break
		}
		err = m.store.Atomic(func(tx crud.Tx) error {
			for _, row := range rows {
				msg := row.(*Message)
				if err := tx.Delete(msg, msg.Id); err != nil {
					return err
				}
				count++
			}
			return nil
		})
		if err != nil {
			return count, err
		}
	}
	return count, nil
}
