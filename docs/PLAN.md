---
PLAN: "feat: export ui.Screens, drop layout dependency, resolve langc reuse warnings"
TAG: v0.2.104
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.

## Tools

```
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
go install webtyp.com/ormc/cmd/ormc@latest
```

## Why

Phase M of webtyp app composition (`webtyp/docs/APP_COMPOSITION_MASTER_PLAN.md`): modules declare their screens via `Screens(caller router.Caller) []view.Screen` instead of legacy `Browser(...) (platformd.UIModule, error)`, and drop `webtyp.com/layout` from `go.mod`.
Additionally, `langc sync .` detected reuse warnings:
1. `chat_room: Deps.IDs is required` and `chat_room: Deps.Participants is required` share `is required`.
2. `other user id` and `user id` share `user id`.
We resolve these via a `depErr` helper in `module.go`, explicit distinct labels in `model.go`, update `view.go` with `opts ...view.Option`, replace `ui/browser.go` with `Screens(caller router.Caller) []view.Screen`, drop `webtyp.com/layout` from `go.mod`, and verify 100% Spanish translations.

## APIs used

| symbol | exact signature | tag | file |
|---|---|---|---|
| `view.Screen` | `type Screen struct { ID string; Label lang.Text; Icon string; Permission model.Resource; Presenter Presenter; Lookups []Lookup; Help lang.Text; Children []Screen }` | `webtyp.com/view@v0.6.45` | `view/screen.go` |
| `view.WithDescription` | `func WithDescription(d lang.Text) Option` | `webtyp.com/view@v0.6.45` | `view/options.go` |
| `view.WithTitle` | `func WithTitle(t string) Option` | `webtyp.com/view@v0.6.45` | `view/options.go` |
| `view.New` | `func New(source Lister, target model.Model, opts ...Option) Presenter` | `webtyp.com/view@v0.6.45` | `view/view.go` |
| `router.Caller` | `type Caller interface { Call(op string, args, out any, done func(error)) }` | `webtyp.com/router@v0.4.8` | `router/caller.go` |

If a signature in the published tag differs from this table, stop and name the symbol in the PR; never adapt it by guessing.

## Stage 1: Explicit labels in model.go and depErr helper in module.go

1. In `model.go`:
- Define constant:
```go
const ResourceGroup = "chat_group"
```
- In `OpenDirectArgsModel`, add explicit label to `other_user_id`:
```go
var OpenDirectArgsModel = model.Definition{
	Name: "open_direct_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "other_user_id", Type: model.Text(), Label: "Contact", NotNull: true},
	},
}
```
- In `RoomParticipantModel` and `PresenceModel`, assign explicit label to `user_id`:
```go
		{Name: "user_id", Type: model.Text(), Label: "User", DB: &model.FieldDB{PK: true}, NotNull: true},
```
- In `AddParticipantArgsModel`:
```go
		{Name: "user_id", Type: model.Text(), Label: "User", NotNull: true},
```
- Run `ormc` to update `model_orm.go`.

2. In `module.go`:
Add helper:
```go
func depErr(name string) error {
	return fmt.Err("chat_room: Deps."+name, "is required")
}
```
And replace the validation lines in `New`:
```go
	if deps.IDs == nil {
		return nil, depErr("IDs")
	}
	if deps.Participants == nil {
		return nil, depErr("Participants")
	}
	if deps.TenantID == "" {
		return nil, depErr("TenantID")
	}
```

Files: `model.go`, `model_orm.go`, `module.go`

## Stage 2: Update view.go with options

In `view.go`:
Update `NewGroupView` to take `opts ...view.Option`:
```go
const titleGroups = "Grupos"

func NewGroupView(caller router.Caller, opts ...view.Option) view.Presenter {
	ops := view.Ops{
		Module: ModelName,
		List:   OpListGroups,
		Save:   OpSaveGroup,
		Delete: OpDeleteGroup,
	}
	lister := view.NewCallerLister(caller, ops, func() model.ModelSlice {
		return &RoomList{}
	})
	allOpts := append([]view.Option{view.WithTitle(titleGroups)}, opts...)
	return view.New(lister, &Room{}, allOpts...)
}
```

Files: `view.go`

## Stage 3: Declarative Screens and drop layout from go.mod

1. In `ui/browser.go`:
Replace with:
```go
//go:build browser

package ui

import (
	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/lang"
	"webtyp.com/router"
	"webtyp.com/view"
)

const (
	ID    = "chat_room"
	Label = "Chat"
)

const (
	descChatGroups lang.Text = "Chat channels, groups and direct messaging rooms."
)

// Screens declara las pantallas que este módulo ofrece al menú de la app.
func Screens(caller router.Caller) []view.Screen {
	return []view.Screen{
		{
			ID:         ID,
			Label:      Label,
			Icon:       ID,
			Permission: chatroom.ResourceGroup,
			Presenter:  chatroom.NewGroupView(caller, view.WithDescription(descChatGroups)),
		},
	}
}
```

2. In `ui/groups.go`:
Remove the legacy `GroupsBrowser` function (or if keeping helper components, remove unused `crudview`/`platformd` imports).
In `ui/source.go`: remove `chatview` import if no longer needed.
In `web/browser.go`: rewrite demo to use `ui.Screens(caller)` and standard `dom`/`html` without importing `platformd`.
```go
//go:build browser

package main

import (
	"github.com/veltylabs/chat_room/ui"
	. "webtyp.com/dom"
	"webtyp.com/html"
	"webtyp.com/router/loopback"
)

func main() {
	caller := loopback.New()
	screens := ui.Screens(caller)
	appNode := html.Div().Child(
		html.H1().Text(string(screens[0].Label)),
		html.P().Text("Demo"),
	)
	Append("body", appNode)
	select {}
}
```

3. Remove `webtyp.com/layout` from `go.mod`. Run `go mod tidy`.

Files: `ui/browser.go`, `ui/groups.go`, `ui/source.go`, `web/browser.go`, `go.mod`, `go.sum`

## Stage 4: Test and Translations

1. In `tests/`:
Update `tests/ui_live_browser_test.go` or replace with `tests/ui_screens_test.go`:
```go
package tests

import (
	"testing"

	"github.com/veltylabs/chat_room/ui"
	"webtyp.com/router/mock"
)

func TestUIScreens(t *testing.T) {
	caller := mock.NewCaller()
	screens := ui.Screens(caller)
	if len(screens) == 0 {
		t.Fatal("expected at least 1 screen")
	}
	for _, s := range screens {
		if err := s.Validate(); err != nil {
			t.Fatalf("screen %s invalid: %v", s.ID, err)
		}
		if s.Presenter == nil {
			t.Fatalf("screen %s expected Presenter", s.ID)
		}
		if s.Presenter.Description() == "" {
			t.Fatalf("screen %s expected non-empty description", s.ID)
		}
	}
}
```
If `tests/ui_live_browser_test.go` imported `platformd`, delete it or remove the `platformd` calls.

2. Run `langc sync .`.
3. Complete 100% of Spanish translations `(es)` in `lang.json`.
4. Ensure `langc sync .` reports:
   - 0 untranslated `(es)`
   - 0 reuse warnings
   - 0 identical values

Files: `tests/ui_screens_test.go`, `tests/ui_live_browser_test.go`, `lang.json`

## Delivery check

Run:
```
gotest
langc sync .
```
Both commands must exit 0 with 0 errors, 0 untranslated `(es)`, and 0 reuse warnings.
Commit after every stage. When all checks pass, push and open the PR.

## Stages

| # | Stage | Files |
|---|---|---|
| 1 | Model labels, ormc and depErr | `model.go`, `model_orm.go`, `module.go` |
| 2 | NewGroupView with options | `view.go` |
| 3 | Declarative screens & drop layout | `ui/browser.go`, `ui/groups.go`, `web/browser.go`, `go.mod`, `go.sum` |
| 4 | Test and translations | `tests/ui_screens_test.go`, `lang.json` |
