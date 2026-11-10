package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

func (p *Member) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	room, err := roomOf(tx, p.RoomId)
	if err != nil {
		return 0, err
	}
	if room.Kind != KindGroup {
		return 0, ErrNotGroup
	}
	if _, ok, err := m.isParticipant(p.UserId); err != nil {
		return 0, err
	} else if !ok {
		return 0, &model.FieldError{Field: Member_.UserId, Err: ErrUnknownUser}
	}
	return crud.ApplyAndPublish, nil
}
