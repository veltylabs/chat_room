//go:build browser

package ui

import (
	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/layout/crudview"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

func Screens(caller router.Caller, ids model.IDGenerator, tenantID string, opts ...Option) []view.Screen {
	chatComp, _ := NewChatComponent(caller, tenantID, opts...)

	crudView, _ := crudview.New(crudview.Config{
		ParentID:  GroupsID + ".list",
		Presenter: chatroom.NewGroupView(caller),
		IDs:       ids,
	})
	mTab := newMembersTab(caller, tenantID)

	return []view.Screen{
		{
			ID:         ID,
			Label:      DefaultLabel,
			Icon:       string(Icon(ID)),
			Component:  chatComp,
			Help:       "Chat interno",
		},
		{
			ID:       GroupsID,
			Label:    GroupsLabel,
			Icon:     string(Icon(GroupsID)),
			Help:     "Administrar grupos",
			Children: []view.Screen{
				{
					ID:         GroupsID + ".list",
					Label:      "Grupos",
					Icon:       string(Icon(GroupsID)),
					Component:  crudView,
					Permission: chatroom.ResourceGroup,
					Help:       "Administrar grupos de chat",
				},
				{
					ID:         GroupsID + ".members",
					Label:      "Miembros",
					Icon:       string(Icon(GroupsID)),
					Component:  mTab,
					Permission: chatroom.ResourceGroup,
					Help:       "Administrar miembros de grupos de chat",
				},
			},
		},
	}
}
