package chatroom

import "webtyp.com/crud"

func (p *Member) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	room, err := roomOf(tx, p.RoomId)
	if err != nil {
		return 0, err
	}
	if room.Kind != KindGroup {
		return 0, ErrNotGroup
	}
	return crud.ApplyAndPublish, nil
}
