package chatroom

import "webtyp.com/crud"

func (p *Room) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	name, err := validGroupName(p.Name)
	if err != nil {
		return crud.RuleAnswer{}, err
	}
	p.Kind = KindGroup
	p.Name = name
	p.DirectKey = ""
	return crud.ApplyAndPublish(), nil
}
