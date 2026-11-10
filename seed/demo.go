//go:build wasm

package seed

import (
	chatroom "github.com/veltylabs/chat_room"
	"github.com/veltylabs/chat_room/migrate"
	"github.com/veltylabs/chat_room/ui"
	"webtyp.com/crud"
	"webtyp.com/ddl"
	"webtyp.com/fmt"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
	"webtyp.com/unixid"
	"webtyp.com/view"
)

const demoTenantID = "tenant-demo"

type demoParticipants struct{}

func (demoParticipants) ParticipantOptions(string) ([]fmt.KeyValue, error) {
	return []fmt.KeyValue{{Key: "u1", Value: "Ana (tú)"}, {Key: "u2", Value: "Bruno"}, {Key: "u3", Value: "Carla"}}, nil
}

// Demo crea una demostración ejecutable del módulo chat_room sobre una base en memoria con datos semilla.
//
// Ejemplo: demo, err := seed.Demo()
func Demo() (view.Demo, error) {
	db := orm.New(mem.New())
	ids, err := unixid.NewUnixID()
	if err != nil {
		return view.Demo{}, err
	}
	mod, err := chatroom.New(db, chatroom.Deps{
		IDs: ids, Participants: demoParticipants{}, TenantID: demoTenantID, RetentionDays: 30,
	})
	if err != nil {
		return view.Demo{}, err
	}
	if _, err := Load(mod, demoTenantID, []string{"u1", "u2"}); err != nil {
		return view.Demo{}, err
	}
	return view.Demo{
		Label:      ui.DefaultLabel,
		Timezone:   "America/Santiago",
		ActingAs:   "u1",
		Modules:    []crud.Module{mod},
		Migrations: []ddl.Step{{Name: chatroom.ModelName, Run: migrate.Migrate}},
		Screens: func(d view.BrowserDeps) []view.Screen {
			return ui.Screens(d.Caller, d.Zone, ids, demoTenantID)
		},
	}, nil
}
