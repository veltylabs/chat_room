package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
	"webtyp.com/time"
)

func isOnline(now, lastSeen time.Nano) bool {
	if lastSeen <= 0 {
		return false
	}
	return now-lastSeen < presenceWindow.Nano()
}

func (m *Module) getPresences() ([]Presence, error) {
	rows, err := allRows(m.store, func() model.Model { return &Presence{} }, nil)
	if err != nil {
		return nil, err
	}
	presences := make([]Presence, len(rows))
	for i, r := range rows {
		presences[i] = *r.(*Presence)
	}
	return presences, nil
}

func (m *Module) Heartbeat(tenantID, userID string) ([]Participant, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	if tenantID == "" {
		tenantID = m.tenantID
	}
	if tenantID != m.tenantID {
		return nil, ErrNotFound
	}
	if _, ok, err := m.isParticipant(userID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrUnknownUser
	}

	now := m.nowNano()
	err := m.store.Atomic(func(tx crud.Tx) error {
		row, found, err := firstRow(tx, func() model.Model { return &Presence{} }, storage.Eq(Presence_.UserId, userID))
		if err != nil {
			return err
		}
		if found {
			presence := row.(*Presence)
			presence.LastSeen = now
			return tx.UpdateFields(presence, presence.UserId, []string{Presence_.LastSeen})
		}
		return tx.Create(&Presence{UserId: userID, LastSeen: now})
	})
	if err != nil {
		return nil, err
	}

	return m.ListParticipants(tenantID, userID)
}
