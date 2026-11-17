package chatroom

import "webtyp.com/crud"

func (p *Room) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if p.Kind != KindGroup {
		return crud.RuleAnswer{}, ErrNotGroup
	}
	if err := deleteGroupRows(tx, p.Id); err != nil {
		return crud.RuleAnswer{}, err
	}
	return crud.ApplyAndPublish(), nil
}
