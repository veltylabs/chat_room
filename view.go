package chatroom

import (
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)


func NewGroupView(caller router.Caller) view.Presenter {
	ops := view.Ops{
		Module: ModelName,
		List:   OpListGroups,
		Save:   OpSaveGroup,
		Delete: OpDeleteGroup,
	}
	lister := view.NewCallerLister(caller, ops, func() model.ModelSlice {
		return &RoomList{}
	})
	return view.New(lister, &Room{})
}
