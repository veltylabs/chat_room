package chatroom

import "webtyp.com/crud"

func (p *Member) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	room, err := roomOf(tx, p.RoomId)
	if err != nil {
		return crud.RuleAnswer{}, err
	}
	if room.Kind != KindGroup {
		return crud.RuleAnswer{}, ErrNotGroup
	}
	return crud.ApplyAndPublish(), nil
}
