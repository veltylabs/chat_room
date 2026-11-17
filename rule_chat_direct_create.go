package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

func (p *Direct) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if p.UserA >= p.UserB {
		return crud.RuleAnswer{}, &model.FieldError{Field: Direct_.UserA, Err: ErrDirectPairInvalid}
	}
	if a.UserID != p.UserA && a.UserID != p.UserB {
		return crud.RuleAnswer{}, &model.FieldError{Field: Direct_.UserA, Err: ErrNotMember}
	}
	other := p.UserA
	if a.UserID == p.UserA {
		other = p.UserB
	}
	if _, ok, err := m.isParticipant(other); err != nil {
		return crud.RuleAnswer{}, err
	} else if !ok {
		return crud.RuleAnswer{}, &model.FieldError{Field: Direct_.UserB, Err: ErrUnknownUser}
	}
	now := m.nowNano()
	room := Room{Id: m.ids.NewID(), Kind: KindDirect, DirectKey: makeDirectKey(p.UserA, p.UserB), CreatedAt: now}
	if err := tx.Create(&room); err != nil {
		return crud.RuleAnswer{}, err
	}
	for _, user := range []string{p.UserA, p.UserB} {
		if err := tx.Create(&Member{RoomId: room.Id, UserId: user}); err != nil {
			return crud.RuleAnswer{}, err
		}
	}
	p.RoomId = room.Id
	p.CreatedAt = now
	return crud.ApplyAndPublish(), nil
}
