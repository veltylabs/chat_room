package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

func (next *Presence) Update(a crud.CurrentUser, tx crud.Tx, old *Presence, fields []string, m *Module) (crud.RuleAnswer, error) {
	if old.UserId != a.UserID {
		return crud.RuleAnswer{}, &model.FieldError{Field: Presence_.UserId, Err: ErrUnauthenticated}
	}
	if err := onlyFields(fields, Presence_.LastSeen); err != nil {
		return crud.RuleAnswer{}, err
	}
	next.LastSeen = m.nowNano()
	return crud.ApplyAndPublish(), nil
}
