← [Stage 4](PLAN_STAGE_4_READERS.md) | Next → [Stage 6](PLAN_STAGE_6_TESTS.md)

# Stage 5 — `view.go`, demo

This module has NO `module/` package: do not create one. `DefaultGrants` (Stage 1) is exported for the app's composition root.

## 5.1 `view.go`

`view.Ops` no longer has `List`, `Save`, `Delete` or `Allow`; the operation is derived from the module and the entity. Replace the whole file with exactly:

```go
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
```
If `view.NewCallerLister` or `view.Ops` differ from this in `view@v0.6.74`, run `go doc webtyp.com/view Ops` and `go doc webtyp.com/view NewCallerLister` and name the difference in the PR; copy https://github.com/veltylabs/patient_directory/blob/main/view.go.

## 5.2 Demo (replaces `web/browser.go`)

Create `seed/demo.go` (`//go:build wasm`), written exactly, then `rm -r web` (`test ! -e web`):

```go
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
		Label:      ui.Label,
		Timezone:   "America/Santiago",
		ActingAs:   "u1",
		Modules:    []crud.Module{mod},
		Migrations: []ddl.Step{{Name: chatroom.ModelName, Run: migrate.Migrate}},
		Screens: func(d view.BrowserDeps) []view.Screen {
			return ui.Screens(d.Caller)
		},
	}, nil
}
```
`ui.Label` and `ui.Screens(caller)`: read `ui/module.go` and `ui/browser.go`; if the label constant or the screens function has another name there, use the existing one (do not add a new one). If `ui.Screens` needs more arguments than the caller (read what the old `web/browser.go` passed it), pass the same.

## 5.3 Greps (all empty)

`grep -rn "webtyp.com/layout\|appkit.BrowserDeps\|appkit.ModuleBrowser\|router/loopback" --include=*.go . | grep -v "^./ui/"`. `ui/` keeps its `webtyp.com/layout/chatview` and `crudview` imports until phase L6 of the architecture master (it builds that part of the chassis): do not touch `ui/`.

`gotest` is green on every destination now (`wasm` and `browser` included) except the tests that Stage 6 rewrites. Commit and push.
