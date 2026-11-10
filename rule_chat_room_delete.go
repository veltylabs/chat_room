package chatroom

import "webtyp.com/crud"

func (p *Room) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if p.Kind != KindGroup {
		return 0, ErrNotGroup
	}
	if err := deleteGroupRows(tx, p.Id); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
