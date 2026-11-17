package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/storage"
)

func (p *Room) Search(a crud.CurrentUser, filter *Room, m *Module) ([]storage.Condition, error) {
	var conditions []storage.Condition
	if filter != nil && filter.Kind != "" {
		conditions = append(conditions, storage.Eq(Room_.Kind, filter.Kind))
	}
	return conditions, nil
}
