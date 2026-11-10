package chatroom

import "webtyp.com/crud"

func (next *Room) Update(a crud.CurrentUser, tx crud.Tx, old *Room, fields []string, m *Module) (crud.RuleAnswer, error) {
	if old.Kind != KindGroup {
		return 0, ErrNotGroup
	}
	if err := onlyFields(fields, Room_.Name); err != nil {
		return 0, err
	}
	name, err := validGroupName(next.Name)
	if err != nil {
		return 0, err
	}
	next.Name = name
	return crud.ApplyAndPublish, nil
}
