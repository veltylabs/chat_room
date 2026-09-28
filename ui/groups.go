package ui

import (
	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/components/decktabs"
	"webtyp.com/dom"
	"webtyp.com/html"
	"webtyp.com/layout/crudview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/svg"
)

func GroupsBrowser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error) {
	crudView, err := crudview.New(crudview.Config{
		ParentID:  GroupsID + ".list",
		Presenter: chatroom.NewGroupView(caller),
		IDs:       ids,
	})
	if err != nil {
		return nil, err
	}

	mTab := newMembersTab(caller, tenantID)

	dt := &decktabs.DeckTabs{
		Items: []decktabs.Item{
			{
				ID:    GroupsID + ".groups",
				Label: "Grupos",
				Panel: crudView,
			},
			{
				ID:    GroupsID + ".members",
				Label: "Miembros",
				Panel: mTab,
			},
		},
	}

	m := &groupsModule{
		id:    GroupsID,
		label: GroupsLabel,
		view:  dt,
	}

	return m, nil
}

type groupsModule struct {
	id, label string
	view      *decktabs.DeckTabs
}

func (m *groupsModule) ModelName() string { return m.id }
func (m *groupsModule) Label() string     { return m.label }
func (m *groupsModule) Icon() svg.Icon    { return Icon(m.id) }
func (m *groupsModule) View() dom.Component { return m.view }

var _ platformd.UIModule = (*groupsModule)(nil)

type membersTab struct {
	dom.Element
	caller    router.Caller
	tenantID  string
	groupSel  *dom.SignalString
	groupOpts *dom.SignalNodes
	boxes     *dom.SignalNodes
	msg       *dom.SignalString
	on        []string
	people    []chatroom.Participant
	selectEl  *dom.Element
}

func newMembersTab(caller router.Caller, tenantID string) *membersTab {
	gSel := dom.NewString("")
	gOpts := dom.NewNodes()
	boxes := dom.NewNodes()
	msg := dom.NewString("")

	t := &membersTab{
		caller:    caller,
		tenantID:  tenantID,
		groupSel:  gSel,
		groupOpts: gOpts,
		boxes:     boxes,
		msg:       msg,
	}

	t.selectEl = dom.NewElement("select").
		BindChildren(gOpts).
		OnChange(func(e dom.Event) {
			t.groupSel.Set(e.TargetValue())
			t.loadMembers()
		})

	return t
}

func (t *membersTab) Init(ctx dom.Ctx) {
	var resGroups chatroom.RoomList
	t.caller.Call(chatroom.ModelName+"."+chatroom.OpListGroups, &chatroom.ListGroupsArgs{TenantId: t.tenantID}, &resGroups, func(err error) {
		if err != nil {
			t.msg.Set(err.Error())
			return
		}

		if resGroups.Len() == 0 {
			t.boxes.Set([]*dom.Element{dom.NewElement("div").Text("Cree un grupo en la pestaña Grupos.")})
			return
		}

		var optNodes []*dom.Element
		for i := 0; i < resGroups.Len(); i++ {
			g := resGroups.At(i).(*chatroom.Room)
			opt := html.Option(g.Id, g.Name)
			if i == 0 {
				t.groupSel.Set(g.Id)
				opt.Attr("selected", "selected")
			}
			optNodes = append(optNodes, opt)
		}
		t.groupOpts.Set(optNodes)

		var resCandidates chatroom.ParticipantList
		t.caller.Call(chatroom.ModelName+"."+chatroom.OpListGroupCandidates, &chatroom.ListRoomsArgs{TenantId: t.tenantID}, &resCandidates, func(err error) {
			if err != nil {
				t.msg.Set(err.Error())
				return
			}
			t.people = make([]chatroom.Participant, resCandidates.Len())
			for i := 0; i < resCandidates.Len(); i++ {
				p := resCandidates.At(i).(*chatroom.Participant)
				t.people[i] = *p
			}
			t.loadMembers()
		})
	})
}

func (t *membersTab) loadMembers() {
	groupID := t.groupSel.Get()
	if groupID == "" {
		t.boxes.Set([]*dom.Element{dom.NewElement("div").Text("Cree un grupo en la pestaña Grupos.")})
		return
	}

	var resMembers chatroom.ParticipantList
	t.caller.Call(chatroom.ModelName+"."+chatroom.OpListGroupMembers, &chatroom.ListGroupMembersArgs{TenantId: t.tenantID, RoomId: groupID}, &resMembers, func(err error) {
		if err != nil {
			t.msg.Set(err.Error())
			return
		}

		t.on = make([]string, resMembers.Len())
		for i := 0; i < resMembers.Len(); i++ {
			p := resMembers.At(i).(*chatroom.Participant)
			t.on[i] = p.UserId
		}

		var boxNodes []*dom.Element
		for _, p := range t.people {
			pID := p.UserId
			chk := html.Input("checkbox").Attr("data-user-id", pID)
			if containsStr(t.on, pID) {
				chk.Attr("checked", "checked")
			}
			chk.OnChange(func(e dom.Event) {
				if e.TargetChecked() {
					if !containsStr(t.on, pID) {
						t.on = append(t.on, pID)
					}
				} else {
					t.on = removeStr(t.on, pID)
				}
			})
			lbl := html.Label().Child(chk, dom.NewElement("span").Text(" "+p.Label))
			boxNodes = append(boxNodes, lbl)
		}
		t.boxes.Set(boxNodes)
	})
}

func (t *membersTab) save() {
	groupID := t.groupSel.Get()
	if groupID == "" {
		return
	}

	idRefs := make([]chatroom.IdRef, len(t.on))
	for i, u := range t.on {
		idRefs[i] = chatroom.IdRef{Id: u}
	}

	args := chatroom.SetGroupMembersArgs{
		TenantId: t.tenantID,
		RoomId:   groupID,
		UserIds:  idRefs,
	}

	t.caller.Call(chatroom.ModelName+"."+chatroom.OpSetGroupMembers, &args, nil, func(err error) {
		if err != nil {
			t.msg.Set(err.Error())
		} else {
			t.msg.Set("Miembros guardados.")
		}
	})
}

func (t *membersTab) Render() *dom.Element {
	return dom.NewElement("div").Class("members-tab").Child(
		dom.NewElement("div").Class("field").Child(
			html.Label().Text("Seleccione un grupo"),
			t.selectEl,
		),
		dom.NewElement("div").Class("checkboxes").BindChildren(t.boxes),
		dom.NewElement("div").Class("actions").Child(
			html.Button().Text("Guardar").OnClick(func(e dom.Event) {
				t.save()
			}),
			dom.NewElement("span").BindText(t.msg),
		),
	)
}

func containsStr(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeStr(slice []string, s string) []string {
	out := make([]string, 0, len(slice))
	for _, item := range slice {
		if item != s {
			out = append(out, item)
		}
	}
	return out
}
