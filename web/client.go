//go:build wasm

package main

import (
	chatroom "github.com/veltylabs/chat_room"
	"github.com/veltylabs/chat_room/seed"
	"github.com/veltylabs/chat_room/ui"
	"webtyp.com/components/themetoggle"
	"webtyp.com/dom"
	"webtyp.com/events/mock"
	"webtyp.com/fmt"
	"webtyp.com/layout/platformd"
	"webtyp.com/orm"
	"webtyp.com/router/loopback"
	"webtyp.com/storage/mem"
	"webtyp.com/time"
	"webtyp.com/unixid"
)

const demoTenantID = "tenant-demo"

type demoUser struct{}

func (demoUser) UserName() string    { return "Ana" }
func (demoUser) UserAvatar() string  { return "" }
func (demoUser) UserRoles() []string { return []string{"Demostración"} }

type demoParticipants struct{}

func (demoParticipants) ParticipantOptions(tenantID string) ([]fmt.KeyValue, error) {
	return []fmt.KeyValue{
		{Key: "u1", Value: "Ana (tú)"},
		{Key: "u2", Value: "Bruno"},
		{Key: "u3", Value: "Carla"},
	}, nil
}

func monotonic() func() int64 {
	var last int64
	return func() int64 {
		now := time.Now()
		if now <= last {
			now = last + 1
		}
		last = now
		return now
	}
}

func main() {
	ids, err := unixid.NewUnixID()
	if err != nil {
		panic(err)
	}

	broker := &mock.Broker{}

	db := orm.New(mem.New())

	mod, err := chatroom.New(db, chatroom.Deps{
		IDs:           ids,
		Participants:  demoParticipants{},
		TenantID:      demoTenantID,
		RetentionDays: 90,
		Publisher:     broker,
		Clock:         monotonic(),
	})
	if err != nil {
		panic(err)
	}

	data, err := seed.Load(mod, demoTenantID, []string{"u1", "u2", "u3"})
	if err != nil {
		panic(err)
	}

	caller := loopback.ActingAs(demoTenantID, "u1", mod)

	chatMod, err := ui.Browser(caller, ids, demoTenantID, ui.WithInbox(broker, "u1"))
	if err != nil {
		panic(err)
	}

	p := &platformd.Platform{
		AppName: ui.DefaultLabel + " — demo",
		User:    demoUser{},
		UserActions: func() dom.Component {
			return &themetoggle.ThemeToggle{}
		},
		Modules:   []platformd.UIModule{chatMod},
		DefaultID: ui.ID,
	}

	_ = dom.Append("body", p)

	time.AfterFunc(8000, func() {
		_, _ = mod.SendMessage(demoTenantID, "u2", data.DirectRoom.RoomId, "¿Tienes un momento?")
	})

	select {}
}
