---
PLAN: "feat(ui)!: chat group administration is a platformd group (Grupos, Miembros); Browser is the chat only — decktabs is gone from components"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 17658227601554359146
---

> Read AGENTS.md at the repo root before any change.

Part of the cross-library UI-consistency wave: libraries published `widget` v0.6.43, `components` v0.8.34,
`form` v0.4.43 and `layout` v0.3.52; this plan moves the module onto them.

## Tools

```
go install webtyp.com/devflow/cmd/gotest@latest
```

## Why

`webtyp.com/components` v0.8.31 **deleted the `decktabs` package**: grouping screens under one navigation item
is the navigation shell's job. `ui/browser.go` and `ui/groups.go` still import it, so `go mod tidy` fails:

```
module webtyp.com/components@latest found (v0.8.34), but does not contain package webtyp.com/components/decktabs
```

The shell's answer is `platformd.NewGroupUIModule` (one navigation entry holding child modules, each with its
own screen); reference implementation:
[`item_catalog/ui/browser.go`](https://github.com/veltylabs/item_catalog/blob/main/ui/browser.go).

Today the module offers the same screens two ways: `Browser` (without `WithoutGroups()`) embeds a three-tab
screen Chat/Grupos/Miembros, and `GroupsBrowser` is a second two-tab screen Grupos/Miembros. The app
(`mjosefa-cms/modules/browser.go`) already uses the separate form (`WithoutGroups()` + `GroupsBrowser`). One
intent, one path: the embedded three-tab screen is deleted.

## Design gate

1. **Prior art:** `item_catalog/ui/browser.go`; `mjosefa-cms/modules/browser.go` (the composition that already
   exists in production).
2. **Novice-name test:** nothing new is named.
3. **Complexity ledger:** −1 exported option (`WithoutGroups`), −1 type (`groupsModule`), −1 field
   (`screen.tabs`), −2 imports of `decktabs`.
4. **Where it belongs:** here; the shell owns grouping.
5. **What it deletes:** `WithoutGroups` and `options.noGroups`; the embedded three-tab branch of `Browser`; the
   `groupsModule` type; every use of `decktabs`. Breaking change: the only consumer is
   `mjosefa-cms/modules/browser.go:69` (`chatOpts := []chatroomui.Option{chatroomui.WithoutGroups()}`), which
   drops the option in its own plan after this one is published.

## Stage 1 — dependencies

```
go get webtyp.com/layout@v0.3.52 webtyp.com/components@v0.8.34 webtyp.com/form@v0.4.43 webtyp.com/widget@v0.6.43
go mod tidy
```

(`go mod tidy` only succeeds after Stages 2 and 3 remove the `decktabs` imports; run it last.)

## Stage 2 — `ui/browser.go`: `Browser` is the chat only

- Delete `WithoutGroups`, the `noGroups` field of `options`, and the whole `if !o.noGroups && caller != nil && ids != nil { … }`
  block that builds `crudView`, `mTab` and `screenTabs`.
- `screen` loses its `tabs` field; `screen.Render` always renders `s.m.view`:

```go
func (s *screen) Render() *dom.Element {
	return dom.NewElement("div").
		Attr("data-chat-screen", "").
		Attr("style", "height:100%;min-height:0;display:flex;flex-direction:column;overflow:hidden;").
		Child(s.m.view)
}
```

  (Keep the rest of `screen` — `Init`, `tick`, the timer — unchanged. Keep the existing `Attr("style", …)`
  exactly as it is: restyling it is out of scope.)
- `m.screen = &screen{m: m}`.
- Delete the imports that become unused (`decktabs`, `crudview` and anything else the compiler reports). `Browser`
  keeps its signature; `ids` stays a parameter (the `model.IDGenerator` is still part of the module-builder
  contract used by the app).

## Stage 3 — `ui/groups.go`: `GroupsBrowser` returns a group

Keep the `crudview.New` call and `newMembersTab` exactly as they are. Replace the `decktabs` composition, the
`groupsModule` type and its methods (and the `var _ platformd.UIModule = (*groupsModule)(nil)` line) with:

```go
groupsMod := platformd.NewUIModule(GroupsID+".list", "Grupos", Icon(GroupsID), crudView)
membersMod := platformd.NewUIModule(GroupsID+".members", "Miembros", Icon(GroupsID), mTab)

return platformd.NewGroupUIModule(GroupsID, GroupsLabel, Icon(GroupsID), []platformd.UIModule{
	groupsMod, membersMod,
}), nil
```

- Both children reuse `Icon(GroupsID)` (already registered in `ui/svg.go`); add no artwork.
- In `membersTab`, the two empty-state texts say "Cree un grupo en la pestaña Grupos." — there are no tabs
  anymore: change both to "Cree un grupo en la sección Grupos." (they are one literal each; if the same literal
  appears twice, introduce one named constant).
- Delete the `decktabs` and unused imports.

## Stage 4 — tests and docs

- `tests/ui_live_wasm_test.go`, section "4. Groups screen": `GroupsBrowser` now returns a group, whose
  `View()` is `nil` by design. Keep the `ModelName() != ui.GroupsID` assertion, add that
  `groupsMod.(platformd.GroupModule).Children()` has exactly two modules with `ModelName()`
  `chat_room.groups.list` and `chat_room.groups.members`, and render the **members child's** `View()` in
  `chat-groups-root` instead of `groupsMod.View()`; the checkbox assertions that follow stay the same. Extend this existing test;
  do not add a second one.
- `docs/ARCHITECTURE.md` and `README.md`: replace every description of tabs with the group, and remove
  `WithoutGroups` from the option list.

## Delivery check

```
grep -rn "decktabs\|WithoutGroups\|groupsModule" . --include=*.go --include=*.md    # → empty
grep -n '^replace' go.mod                                                          # → empty
gotest                                                                             # vet, race, tests and wasm green
git diff --stat main                                                               # only the files named above
```

## Stages

| # | Stage | Files |
|---|---|---|
| 1 | dependencies | `go.mod`, `go.sum` |
| 2 | `Browser` is the chat only | `ui/browser.go` |
| 3 | `GroupsBrowser` is a group | `ui/groups.go` |
| 4 | tests and docs | `tests/ui_live_wasm_test.go`, `docs/ARCHITECTURE.md`, `README.md` |

## Executor notes
- `ui/groups.go` refactored `GroupsBrowser` to use `platformd.NewGroupUIModule` replacing `decktabs`.
- `ui/browser.go` removed `decktabs` and `WithoutGroups` function option.
- Bumps components, layout, widget, form dependencies.
