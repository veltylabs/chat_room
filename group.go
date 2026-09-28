package chatroom

import (
	"webtyp.com/events"
	"webtyp.com/model"
	"webtyp.com/orm"
)

func (m *Module) ListGroups(tenantID string) ([]Room, error) {
	if tenantID == "" {
		tenantID = m.tenantID
	}

	var groups []Room
	err := m.db.Query(&Room{}).
		Where("tenant_id").Eq(tenantID).
		Where("kind").Eq(KindGroup).
		ReadAll(func() model.Model { return &Room{} }, func(m model.Model) {
			groups = append(groups, *m.(*Room))
		})
	if err != nil {
		return nil, err
	}

	return groups, nil
}

func (m *Module) SaveGroup(tenantID, id, name string) (Room, error) {
	if tenantID == "" {
		tenantID = m.tenantID
	}

	trimmedName := trimSpaces(name)
	if len([]rune(trimmedName)) < 1 || len([]rune(trimmedName)) > 60 {
		return Room{}, ErrNotFound
	}

	now := m.nowNano()

	if id != "" {
		var existing Room
		err := m.db.Query(&existing).
			Where("tenant_id").Eq(tenantID).
			Where("id").Eq(id).
			ReadOne()
		if err == orm.ErrNotFound {
			return Room{}, ErrNotFound
		}
		if err != nil {
			return Room{}, err
		}
		if existing.Kind != KindGroup {
			return Room{}, ErrNotGroup
		}

		existing.Name = trimmedName
		existing.UpdatedAt = now
		err = m.db.UpdateFields(&existing, []string{"name", "updated_at"},
			orm.Eq(Room_.TenantId, tenantID),
			orm.Eq(Room_.Id, id),
		)
		if err != nil {
			return Room{}, err
		}

		if m.publisher != nil {
			m.publisher.Publish(events.Event{
				Topic:   TopicGroupSaved,
				Payload: &existing,
			})
		}

		return existing, nil
	}

	groupID := m.ids.NewID()
	group := Room{
		Id:        groupID,
		TenantId:  tenantID,
		Kind:      KindGroup,
		Name:      trimmedName,
		CreatedAt: now,
	}

	err := m.db.Create(&group)
	if err != nil {
		return Room{}, err
	}

	if m.publisher != nil {
		m.publisher.Publish(events.Event{
			Topic:   TopicGroupSaved,
			Payload: &group,
		})
	}

	return group, nil
}

func (m *Module) SetGroupMembers(tenantID, roomID string, userIDs []string) error {
	if tenantID == "" {
		tenantID = m.tenantID
	}

	var room Room
	err := m.db.Query(&room).
		Where("tenant_id").Eq(tenantID).
		Where("id").Eq(roomID).
		ReadOne()
	if err == orm.ErrNotFound {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if room.Kind != KindGroup {
		return ErrNotGroup
	}

	opts, err := m.participants.ParticipantOptions(tenantID)
	if err != nil {
		return err
	}

	validUsers := make([]string, 0, len(userIDs))
	for _, uid := range userIDs {
		found := false
		for _, opt := range opts {
			if opt.Key == uid {
				found = true
				break
			}
		}
		if !found {
			return ErrUnknownUser
		}
		validUsers = append(validUsers, uid)
	}

	var existingMembers []Member
	err = m.db.Query(&Member{}).
		Where("tenant_id").Eq(tenantID).
		Where("room_id").Eq(roomID).
		ReadAll(func() model.Model { return &Member{} }, func(m model.Model) {
			existingMembers = append(existingMembers, *m.(*Member))
		})
	if err != nil {
		return err
	}

	readAtMap := make([]fmtKeyValueInt, 0, len(existingMembers))
	existingIDs := make([]string, 0, len(existingMembers))
	for _, em := range existingMembers {
		readAtMap = append(readAtMap, fmtKeyValueInt{Key: em.UserId, Value: em.LastReadAt})
		existingIDs = append(existingIDs, em.UserId)
	}

	var added []string
	var removed []string

	for _, uid := range validUsers {
		inExisting := false
		for _, em := range existingMembers {
			if em.UserId == uid {
				inExisting = true
				break
			}
		}
		if !inExisting {
			added = append(added, uid)
		}
	}

	for _, em := range existingMembers {
		inNew := false
		for _, uid := range validUsers {
			if uid == em.UserId {
				inNew = true
				break
			}
		}
		if !inNew {
			removed = append(removed, em.UserId)
		}
	}

	err = m.db.Tx(func(tx *orm.DB) error {
		for _, em := range existingMembers {
			if err := tx.Delete(&em, orm.Eq(Member_.TenantId, tenantID), orm.Eq(Member_.RoomId, roomID), orm.Eq(Member_.UserId, em.UserId)); err != nil {
				return err
			}
		}

		for _, uid := range validUsers {
			var lastRead int64
			for _, kv := range readAtMap {
				if kv.Key == uid {
					lastRead = kv.Value
					break
				}
			}

			newMem := Member{
				TenantId:   tenantID,
				RoomId:     roomID,
				UserId:     uid,
				LastReadAt: lastRead,
			}
			if err := tx.Create(&newMem); err != nil {
				return err
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

type fmtKeyValueInt struct {
	Key   string
	Value int64
}

func (m *Module) ListGroupCandidates(tenantID string) ([]Participant, error) {
	if tenantID == "" {
		tenantID = m.tenantID
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

	var room Room
	err := m.db.Query(&room).
		Where("tenant_id").Eq(tenantID).
		Where("id").Eq(roomID).
		ReadOne()
	if err == orm.ErrNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if room.Kind != KindGroup {
		return nil, ErrNotGroup
	}

	var members []Member
	err = m.db.Query(&Member{}).
		Where("tenant_id").Eq(tenantID).
		Where("room_id").Eq(roomID).
		ReadAll(func() model.Model { return &Member{} }, func(m model.Model) {
			members = append(members, *m.(*Member))
		})
	if err != nil {
		return nil, err
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
	var participants []Participant

	for _, mem := range members {
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

		var lastSeen int64
		for _, p := range presences {
			if p.UserId == mem.UserId {
				lastSeen = p.LastSeen
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

	var room Room
	err := m.db.Query(&room).
		Where("tenant_id").Eq(tenantID).
		Where("id").Eq(id).
		ReadOne()
	if err == orm.ErrNotFound {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if room.Kind != KindGroup {
		return ErrNotGroup
	}

	var members []Member
	_ = m.db.Query(&Member{}).
		Where("tenant_id").Eq(tenantID).
		Where("room_id").Eq(id).
		ReadAll(func() model.Model { return &Member{} }, func(m model.Model) {
			members = append(members, *m.(*Member))
		})

	var messages []Message
	_ = m.db.Query(&Message{}).
		Where("tenant_id").Eq(tenantID).
		Where("room_id").Eq(id).
		ReadAll(func() model.Model { return &Message{} }, func(m model.Model) {
			messages = append(messages, *m.(*Message))
		})

	err = m.db.Tx(func(tx *orm.DB) error {
		for _, msg := range messages {
			if err := tx.Delete(&msg, orm.Eq(Message_.TenantId, tenantID), orm.Eq(Message_.RoomId, id), orm.Eq(Message_.Id, msg.Id)); err != nil {
				return err
			}
		}
		for _, mem := range members {
			if err := tx.Delete(&mem, orm.Eq(Member_.TenantId, tenantID), orm.Eq(Member_.RoomId, id), orm.Eq(Member_.UserId, mem.UserId)); err != nil {
				return err
			}
		}
		return tx.Delete(&room, orm.Eq(Room_.TenantId, tenantID), orm.Eq(Room_.Id, id))
	})

	return err
}

func (m *Module) PurgeExpired(tenantID string) (int, error) {
	if tenantID == "" {
		tenantID = m.tenantID
	}

	now := m.nowNano()
	retentionNanos := int64(m.retentionDays) * 24 * 3600 * 1_000_000_000
	cutoff := now - retentionNanos

	var expired []Message
	err := m.db.Query(&Message{}).
		Where("tenant_id").Eq(tenantID).
		Where("created_at").Lt(cutoff).
		ReadAll(func() model.Model { return &Message{} }, func(m model.Model) {
			expired = append(expired, *m.(*Message))
		})
	if err != nil {
		return 0, err
	}

	count := 0
	for _, msg := range expired {
		err := m.db.Delete(&msg, orm.Eq(Message_.TenantId, tenantID), orm.Eq(Message_.Id, msg.Id))
		if err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}
