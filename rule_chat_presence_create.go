package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

func (p *Presence) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if p.UserId != a.UserID {
		return 0, &model.FieldError{Field: Presence_.UserId, Err: ErrUnauthenticated}
	}
	if _, ok, err := m.isParticipant(a.UserID); err != nil {
		return 0, err
	} else if !ok {
		return 0, &model.FieldError{Field: Presence_.UserId, Err: ErrUnknownUser}
	}
	p.LastSeen = m.nowNano()
	return crud.ApplyAndPublish, nil
}
