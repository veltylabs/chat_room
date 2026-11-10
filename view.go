package chatroom

import (
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

// NewGroupView builds the presenter of the groups screen over the rooms table.
//
// Example: presenter := chatroom.NewGroupView(caller)
func NewGroupView(caller router.Caller) view.Presenter {
	ops := view.Ops{
		Module: ModelName,
		Page:   view.Page{First: view.All},
	}
	lister := view.NewCallerLister(view.NewRecords().Caller(caller), ops, func() model.TableSlice {
		return &RoomList{}
	})
	return view.New(lister, &Room{})
}
