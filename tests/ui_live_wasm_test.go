//go:build wasm

package tests

import (
	"testing"

	chatroom "github.com/veltylabs/chat_room"
	"github.com/veltylabs/chat_room/seed"
	"github.com/veltylabs/chat_room/ui"
	"webtyp.com/events/mock"
	"webtyp.com/fmt"
	"webtyp.com/layout/platformd"
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

func wasmClock() func() int64 {
	var last int64 = time.Now()
	return func() int64 {
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

	chatMod, err := ui.Browser(caller, ids, tenant, ui.WithInbox(broker, "u1"))
	if err != nil {
		t.Fatalf("ui.Browser failed: %v", err)
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
	groupsMod, err := ui.GroupsBrowser(caller, ids, tenant)
	if err != nil {
		t.Fatalf("ui.GroupsBrowser failed: %v", err)
	}
	if groupsMod.ModelName() != ui.GroupsID {
		t.Fatalf("expected ModelName %q, got %q", ui.GroupsID, groupsMod.ModelName())
	}
}
