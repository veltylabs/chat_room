package chatroom

import (
	"webtyp.com/time"
	"webtyp.com/model"
	"webtyp.com/orm"
)

func isOnline(now, lastSeen time.Nano) bool {
	if lastSeen <= 0 {
		return false
	}
	return now-lastSeen < presenceWindow.Nano()

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
	} else if orm.IsNotFound(err) {
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
