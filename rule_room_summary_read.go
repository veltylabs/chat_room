package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

func (p *RoomSummary) Read(a crud.CurrentUser, filter *RoomSummary, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	rows, err := m.ListRooms(m.tenantID, a.UserID)
	if err != nil {
		return nil, err
	}
	list := RoomSummaryList{}
	for i := range rows {
		row := rows[i]
		list = append(list, &row)
	}
	return &list, nil
}
