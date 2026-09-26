package chatroom

import (
	"webtyp.com/model"
	"webtyp.com/orm"
)

func isOnline(nowNano int64, lastSeenNano int64) bool {
	if lastSeenNano <= 0 {
		return false
	}
	diffSec := (nowNano - lastSeenNano) / 1_000_000_000
	return diffSec < PresenceWindowSeconds
}

func (m *Module) getPresences(tenantID string) ([]Presence, error) {
	var presences []Presence
	err := m.db.Query(&Presence{}).
		Where("tenant_id").Eq(tenantID).
		ReadAll(func() model.Model { return &Presence{} }, func(m model.Model) {
			presences = append(presences, *m.(*Presence))
		})
	if err != nil {
		return nil, err
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

	opts, err := m.participants.ParticipantOptions(tenantID)
	if err != nil {
		return nil, err
	}

	var found bool
	for _, opt := range opts {
		if opt.Key == userID {
			found = true
			break
		}
	}
	if !found {
		return nil, ErrUnknownUser
	}

	now := m.nowNano()

	var p Presence
	err = m.db.Query(&p).
		Where("tenant_id").Eq(tenantID).
		Where("user_id").Eq(userID).
		ReadOne()

	if err == nil {
		p.LastSeen = now
		err = m.db.UpdateFields(&p, []string{"last_seen"},
			orm.Eq(Presence_.TenantId, tenantID),
			orm.Eq(Presence_.UserId, userID),
		)
		if err != nil {
			return nil, err
		}
	} else if err == orm.ErrNotFound {
		newP := Presence{
			TenantId: tenantID,
			UserId:   userID,
			LastSeen: now,
		}
		err = m.db.Create(&newP)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	return m.ListParticipants(tenantID, userID)
}
