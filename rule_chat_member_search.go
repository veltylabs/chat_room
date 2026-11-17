package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/storage"
)

func (p *Member) Search(a crud.CurrentUser, filter *Member, m *Module) ([]storage.Condition, error) {
	var conditions []storage.Condition
	if filter != nil && filter.RoomId != "" {
		conditions = append(conditions, storage.Eq(Member_.RoomId, filter.RoomId))
	}
	return conditions, nil
}
