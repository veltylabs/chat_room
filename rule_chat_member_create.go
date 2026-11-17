package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

func (p *Member) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	room, err := roomOf(tx, p.RoomId)
	if err != nil {
		return crud.RuleAnswer{}, err
	}
	if room.Kind != KindGroup {
		return crud.RuleAnswer{}, ErrNotGroup
	}
	if _, ok, err := m.isParticipant(p.UserId); err != nil {
		return crud.RuleAnswer{}, err
	} else if !ok {
		return crud.RuleAnswer{}, &model.FieldError{Field: Member_.UserId, Err: ErrUnknownUser}
	}
	return crud.ApplyAndPublish(), nil
}
