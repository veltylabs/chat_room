//go:build browser

package tests

import (
	"syscall/js"
	"testing"

	chatroom "github.com/veltylabs/chat_room"
	"github.com/veltylabs/chat_room/seed"
	"github.com/veltylabs/chat_room/ui"
	"webtyp.com/components/inboxlist"
	"webtyp.com/dom"
	"webtyp.com/events/mock"
	"webtyp.com/fmt"
	"webtyp.com/layout/app"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/router/loopback"
	"webtyp.com/storage/mem"
	"webtyp.com/time"
)

type wasmParticipants struct{}

func (wasmParticipants) ParticipantOptions(tenantID string) ([]fmt.KeyValue, error) {
	return []fmt.KeyValue{
		{Key: "u1", Value: "Ana (tú)"},
		{Key: "u2", Value: "Bruno"},
		{Key: "u3", Value: "Carla"},
	}, nil
}

func wasmClock() func() time.Nano {
	last := time.NowNano()
	return func() time.Nano {
		last += 1_000_000
		return last
	}
}

func TestChatScreenLiveWasm(t *testing.T) {
	tenant := "tenant-wasm"
	ids := &mockIDGen{}
	broker := &mock.Broker{}
	db := orm.New(mem.New())

	mod, err := chatroom.New(db, chatroom.Deps{
		IDs:           ids,
		Participants:  wasmParticipants{},
		TenantID:      tenant,
		RetentionDays: 90,
		Publisher:     broker,
		Clock:         wasmClock(),
	})
	if err != nil {
		t.Fatalf("chatroom.New failed: %v", err)
	}

	seedData, err := seed.Load(mod, tenant, []string{"u1", "u2", "u3"})
	if err != nil {
		t.Fatalf("seed.Load failed: %v", err)
	}

	caller := loopback.ActingAs(tenant, "u1", mod)

	screens := ui.Screens(caller, ids, tenant, ui.WithInbox(broker, "u1"))
	menu, err := app.BuildMenu(screens, ids, func(model.Resource) bool { return true }, app.Tools{})
	if err != nil {
		t.Fatalf("app.BuildMenu failed: %v", err)
	}

	// Find Chat and Group modules from menu
	var chatMod, groupsMod platformd.UIModule
	for _, m := range menu.Modules {
		if m.ModelName() == ui.ID {
			chatMod = m
		} else if m.ModelName() == ui.GroupsID {
			groupsMod = m
		}
	}

	if chatMod == nil {
		t.Fatalf("expected chat module in menu")
	}
	if groupsMod == nil {
		t.Fatalf("expected groups module in menu")
	}

	doc := js.Global().Get("document")
	root := doc.Call("createElement", "div")
	root.Set("id", "chat-live-root")
	doc.Get("body").Call("appendChild", root)
	if err := dom.Render("chat-live-root", chatMod.View()); err != nil {
		t.Fatalf("dom.Render: %v", err)
	}

	// Must trigger view load (simulating router load since we are testing components manually here without a full router frame)
	chatMod.Activate()

	// 0. The inbox lists u1's three rooms: General, Recepción, the direct room with Bruno.
	rowSel := "." + string(inboxlist.NameInboxList.Class(inboxlist.PartRow))
	if n := root.Call("querySelectorAll", rowSel).Get("length").Int(); n != 3 {
		t.Fatalf("inbox rows = %d, want 3", n)
	}

	// 1. Unread badge initially
	badged := chatMod.(platformd.Badged)
	badge := badged.Badge()
	if badge.Count.Get() != "1" || !badge.Visible.Get() {
		t.Fatalf("expected unread badge count 1 and visible, got count %q visible %v", badge.Count.Get(), badge.Visible.Get())
	}

	chatViewComp := chatMod.View()
	if chatViewComp == nil {
		t.Fatalf("chatMod.View() returned nil")
	}

	// 2. Push message from u2 refreshes badge count
	_, err = mod.SendMessage(tenant, "u2", seedData.DirectRoom.RoomId, "otro mensaje")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if badge.Count.Get() != "2" {
		t.Fatalf("expected badge count to update to 2 after push, got %q", badge.Count.Get())
	}

	// 3. Heartbeat presence
	parts, err := mod.ListParticipants(tenant, "u2")
	if err != nil {
		t.Fatalf("ListParticipants failed: %v", err)
	}
	foundU1Online := false
	for _, p := range parts {
		if p.UserId == "u1" && p.Online {
			foundU1Online = true
			break
		}
	}
	if !foundU1Online {
		t.Fatalf("expected u1 to be online after heartbeat")
	}

	// 4. Groups screen
	gMod, ok := groupsMod.(platformd.GroupModule)
	if !ok {
		t.Fatalf("expected groupsMod to implement platformd.GroupModule")
	}
	children := gMod.Children()
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
	if children[0].ModelName() != "chat_room.groups.list" {
		t.Fatalf("expected first child ModelName chat_room.groups.list, got %q", children[0].ModelName())
	}
	if children[1].ModelName() != "chat_room.groups.members" {
		t.Fatalf("expected second child ModelName chat_room.groups.members, got %q", children[1].ModelName())
	}

	// Call activate on members to fetch groups properly
	children[1].Activate()

	groot := doc.Call("createElement", "div")
	groot.Set("id", "chat-groups-root")
	doc.Get("body").Call("appendChild", groot)
	if err := dom.Render("chat-groups-root", children[1].View()); err != nil {
		t.Fatalf("dom.Render groups: %v", err)
	}
	// Miembros: every participant (u1 included), Recepción's members (u1, u2) checked.
	boxes := groot.Call("querySelectorAll", "input[type='checkbox'][data-user-id]")
	if n := boxes.Get("length").Int(); n != 3 {
		t.Fatalf("member checkboxes = %d, want 3", n)
	}
	checked := 0
	for i := 0; i < 3; i++ {
		if boxes.Call("item", i).Get("checked").Bool() {
			checked++
		}
	}
	if checked != 2 {
		t.Fatalf("checked members = %d, want 2 (Recepción = u1, u2)", checked)
	}
}
