package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

func (p *MessageView) Read(a crud.CurrentUser, filter *MessageView, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	if filter == nil || filter.RoomId == "" {
		return nil, &model.FieldError{Field: "room_id", Err: ErrRoomRequired}
	}
	rows, err := m.ListMessages(m.tenantID, a.UserID, filter.RoomId, filter.AfterId, int(args.Limit))
	if err != nil {
		return nil, err
	}
	list := MessageViewList{}
	for i := range rows {
		row := rows[i]
		list = append(list, &row)
	}
	return &list, nil
}
