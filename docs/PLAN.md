---
PLAN: "feat(ui): chat_room screen (chatview + push inbox + heartbeat), group admin, runnable demo; schema out of New"
TAG: v0.2.0
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 6283231162488346490
PR: https://github.com/veltylabs/chat_room/pull/2
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.
> Read `AGENTS.md` at the repo root first. Its import rules apply to every file you touch:
> no stdlib `fmt`/`strings`/`strconv`/`errors`/`time` (use `webtyp.com/fmt`, `webtyp.com/time`), no Go maps,
> no `reflect`. **The wasm build decides**: `gotest` runs the wasm suite too, and a change that passes only
> the stdlib build is not done.

# Plan — `chat_room` v0.2.0: the screen

## 0. Context (verified — do not re-diagnose)

`v0.1.0` shipped the domain (rooms, messages, unread, presence, inbox topic, retention) with no UI.
This plan adds `ui/` (the chat screen and the group admin screen) and `web/` (runnable demo), and fixes
three places where the domain package breaks `AGENTS.md`.

Upstream pieces, all published — bump `go.mod` to at least these versions with
`go get webtyp.com/layout@v0.3.3 webtyp.com/components@v0.7.0 webtyp.com/router@v0.1.43` + `go mod tidy`:

| Piece | What you use | Notes |
|---|---|---|
| `webtyp.com/layout/chatview` | `chatview.New(chatview.Config{Source, MaxBodyLength}) (*ChatView, error)`; methods `Refresh()`, `RefreshPeople()`, `Unread() (*dom.SignalString, *dom.SignalBool)`; field `OnError func(error)` | Draws inbox + people tabs + thread + compose bar. You implement its `Source`. `Unread()` is valid right after `New`. |
| `chatview.Source` | see Stage 3 | Every method async, calls `done` exactly once. |
| `webtyp.com/components/{inboxlist,bubblethread,presencelist}` | the row types `inboxlist.Row{ID, Title, Preview, Time, Unread}`, `bubblethread.Bubble{ID, Author, Body, Time, Mine, Read}`, `presencelist.Person{ID, Label, Online}` | Plain structs. |
| `webtyp.com/components/countbadge` | `&countbadge.CountBadge{Count: c, Visible: v}` | The rail badge. |
| `webtyp.com/layout/platformd` | `platformd.UIModule` (`ModelName/Label/Icon/View`); optional `Badged{ Badge() *countbadge.CountBadge }`, `UsesNotifier{ UseNotifier(n platformd.Notifier) }`; `Notifier.Notify(fmt.Msg.Info, msg, platformd.Auto())` | The chassis calls `UseNotifier` once in its `Init`, and renders `Badge()` in the rail. All module views are mounted at start (hidden when not current). |
| `webtyp.com/events` | `events.Subscriber{ Subscribe(topic string, h events.Handler) }` | In the app it is `sse.Subscriber`; in the demo `*mock.Broker`. The event received has **no payload**: it only means "your rooms changed". |
| `webtyp.com/router/loopback` | `loopback.ActingAs(tenantID, userID string, mods...) router.Caller` | New in v0.1.43: in-process caller whose handlers see `ctx.UserID() == userID`. Every chat op is `.Authenticated()`; `loopback.New`/`WithTenant` would get 401. |
| `webtyp.com/layout/crudview` | `crudview.New(crudview.Config{ParentID, Presenter, IDs})` | Group CRUD, over the existing `chatroom.NewGroupView(caller)`. |

Decisions already taken (do not reopen): staff ↔ staff only; rooms `direct`/`group`/`broadcast`; only members
read a room (not even admin); unread counter + "read" on 1:1; no attachments, no "typing…"; presence by
heartbeat every `HeartbeatIntervalSeconds` (30 s), online if the last one is < 90 s old; push = an event
**without content** on `InboxTopic(userID)`, after which the browser asks for data through the normal ops;
rail counter + chassis notification, no sound.

## 1. Rules for this plan

- **Never compose an element id.** `dom` mints them. For hooks, `Attr("data-...", v)`.
- Network calls from `Init(dom.Ctx)` or from callbacks, never from a constructor — except that `Browser`
  may call `Subscribe` (registering a handler is not a network call).
- Every enum/topic/op name is an existing exported constant (`chatroom.OpListRooms`, `chatroom.InboxTopic(...)`,
  `chatroom.HeartbeatIntervalSeconds`, `chatroom.MaxBodyLength`, …). No literal copies.
- Spanish user-facing text in `ui/` (the chatview's own texts come translated from `fmt/lang`).
- Errors from ops reach the user: `ChatView.OnError` → the notifier (`fmt.Msg.Error`), when a notifier is set.

## 2. Stages

### Stage 1 — domain package: align with `AGENTS.md`

1. `module.go`: **delete** the `ddl` block from `New` (the `if ddlCompiler, ok := db.RawConn().(ddl.Compiler)` …
   `d.Sync(...)` lines) and the `webtyp.com/ddl` import. Schema creation belongs only to `migrate/`. Reason:
   `ui/` imports the root package, so `ddl` would be compiled into every browser binary.
2. `migrate/migrate.go`: replace `func Sync(conn storage.Conn) error` with the signature every sibling module uses
   (reference: `github.com/veltylabs/patient_directory/migrate/migrate.go`):
   ```go
   // Migrate reconciles the schema chat_room owns: Room, Member, Message, Presence (in that order: FKs).
   // Deliberately NOT called by New. Sync, not CreateTable: additive, safe to run repeatedly.
   func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error {
   	return ddl.New(conn, ddlCompiler).Sync(&chatroom.Room{}, &chatroom.Member{}, &chatroom.Message{}, &chatroom.Presence{})
   }
   ```
   Update `tests/extra_branches_test.go` (it calls `migrate.Sync`). If `storage/mem` does not implement
   `ddl.Compiler`, that test asserts nothing useful — replace it with a compile-time check only:
   `var _ = migrate.Migrate`.
3. `ops.go`: delete `func (m *Module) MountOps(...)`; move its body into `MountOperations`. One method, one path.
   Update any test that calls `MountOps`.
4. `ops.go` `writeError`: map `ErrTenantRequired` to `400` (today it falls into `500`).
5. New op for the group admin screen (the members editor must list **everyone**, including the admin
   editing — `list_participants` excludes the caller):
   ```go
   OpListGroupCandidates = "list_group_candidates"
   reg.Operation(OpListGroupCandidates, m.handleListGroupCandidates).Requires(ResourceGroup, model.Read).Accepts(&ListRoomsArgs{})
   ```
   Service method `func (m *Module) ListGroupCandidates(tenantID string) ([]Participant, error)`: every
   `ParticipantOptions(tenantID)` entry as `Participant{UserId, Label, Online: false}`, tenant defaulting like
   `ListParticipants`. Handler shape = `handleListGroupMembers`. Tests: returns all users including the one
   calling; tenant B's reader list is not mixed with tenant A's.

Acceptance of Stage 1: `grep -rn "webtyp.com/ddl" --include=*.go . | grep -v "^./migrate/"` → empty;
`grep -rn "MountOps\b" .` → empty; `grep -rn "migrate.Sync" .` → empty.

### Stage 2 — `ui/module.go`

```go
package ui
const (
	ID           = "chat_room"        // UIModule id of the chat screen
	DefaultLabel = "Chat"
	GroupsID     = "chat_room.groups" // UIModule id of the group admin screen
	GroupsLabel  = "Grupos de chat"
)
```
`ui/svg.go` (`//go:build !wasm`): copy `github.com/veltylabs/room_layout/ui/svg.go` (the `Icon(id string) svg.Icon`
helper), and use it for both modules.

### Stage 3 — `ui/source.go`: `chatview.Source` over the ops

```go
type source struct {
	caller   router.Caller
	tenantID string
}
var _ chatview.Source = (*source)(nil)
```

| Method | Op | Args | Mapping |
|---|---|---|---|
| `Rooms(done)` | `OpListRooms` | `ListRoomsArgs{TenantId}` → `RoomSummaryList` | `inboxlist.Row{ID: RoomId, Title, Preview: LastPreview, Time: shortTime(LastMessageAt), Unread: int(Unread)}` |
| `Messages(roomID, done)` | `OpListMessages` | `ListMessagesArgs{TenantId, RoomId}` → `MessageViewList` | `bubblethread.Bubble{ID: Id, Author: SenderLabel, Body, Time: clock(CreatedAt), Mine, Read}` |
| `Send(roomID, body, done)` | `OpSendMessage` | `SendMessageArgs{TenantId, RoomId, Body}` → `MessageView` | same Bubble mapping |
| `MarkRead(roomID, done)` | `OpMarkRead` | `MarkReadArgs{TenantId, RoomId}` | — |
| `People(done)` | **`OpHeartbeat`** | `HeartbeatArgs{TenantId}` → `ParticipantList` | `presencelist.Person{ID: UserId, Label, Online}` |
| `OpenDirect(personID, done)` | `OpOpenDirect` | `OpenDirectArgs{TenantId, OtherUserId: personID}` → `RoomSummary` | `done(summary.RoomId, nil)` |

`People` calls `heartbeat` on purpose: it records the caller's presence **and** returns everyone's, so the
periodic `RefreshPeople()` (Stage 4) is the heartbeat. Op names are qualified: `chatroom.ModelName + "." + chatroom.OpX`.
On error, call `done(nil/zero, err)`.

Time helpers in the same file (nanosecond timestamps; `webtyp.com/time`):
- `clock(ns int64) string` → `"HH:MM"`: `time.FormatTime(ns)` returns `"HH:MM:SS"`; keep the first 5 bytes
  (guard `len >= 5`). `0` → `""`.
- `shortTime(ns int64) string` → `clock(ns)` when `time.IsToday(ns)`, else `time.FormatDate(ns)`; `0` → `""`.
Unit test both with a fixed timestamp (stdlib build).

### Stage 4 — `ui/browser.go`: the chat screen

```go
type Option func(*options)
func WithLabel(label string) Option
// WithInbox turns on push: the screen subscribes to chatroom.InboxTopic(userID) on sub and refreshes when
// it fires. userID must be the signed-in user's id (the same identity the Caller carries).
func WithInbox(sub events.Subscriber, userID string) Option

func Browser(caller router.Caller, ids model.IDGenerator, tenantID string, opts ...Option) (platformd.UIModule, error)
```
(`ids` is unused by the chat screen; it stays for signature parity with every module's `Browser`.)

`Browser`:
1. `v, err := chatview.New(chatview.Config{Source: &source{caller, tenantID}, MaxBodyLength: chatroom.MaxBodyLength})`.
2. `count, visible := v.Unread()`; `badge := &countbadge.CountBadge{Count: count, Visible: visible}`.
3. Build `m := &chatModule{...}` (below). With `WithInbox`: `sub.Subscribe(chatroom.InboxTopic(userID), func(events.Event) { m.onPush() })`.
4. Return `m`.

```go
type chatModule struct {
	id, label string
	view      *chatview.ChatView
	screen    *screen
	badge     *countbadge.CountBadge
	notifier  platformd.Notifier // nil until the chassis calls UseNotifier
	lastUnread string
}
func (m *chatModule) ModelName() string                     { return m.id }   // ID
func (m *chatModule) Label() string
func (m *chatModule) Icon() svg.Icon                        // Icon(ID) — the svg helper; wasm build: see svg.go note below
func (m *chatModule) View() dom.Component                   { return m.screen }
func (m *chatModule) Badge() *countbadge.CountBadge         { return m.badge }
func (m *chatModule) UseNotifier(n platformd.Notifier)      { m.notifier = n }
var _ platformd.Badged = (*chatModule)(nil)
var _ platformd.UsesNotifier = (*chatModule)(nil)
```
(Check how `platformd.NewUIModule`'s `uiModule` implements `Icon()` in both builds and mirror it; if `Icon`
comes from a `!wasm` file, take the icon value in `Browser` exactly as `room_layout/ui/browser.go` does and store it.)

- `onPush()`: `m.view.Refresh()`. After the refresh settles, if the unread count went **up** compared with
  `m.lastUnread` and `m.notifier != nil`, call `m.notifier.Notify(fmt.Msg.Info, "Nuevo mensaje en el chat", platformd.Auto())`.
  Implement "after it settles" without a callback on `Refresh`: read `count.Get()` inside a `dom` effect is
  not available, so compare on the **next** push and on each heartbeat tick instead: keep `lastUnread`, and in
  both places do `now := count.Get(); if atoi(now) > atoi(m.lastUnread) { notify }; m.lastUnread = now`.
  (`loopback` is synchronous, so in the demo and the tests the count is already updated when `Refresh()` returns.)
- `v.OnError = func(err error) { if m.notifier != nil { m.notifier.Notify(fmt.Msg.Error, err.Error(), platformd.Auto()) } }`.

`screen` (same file or `ui/screen.go`): `type screen struct { dom.Element; m *chatModule; timer time.Timer }`
- `Render()`: `dom.NewElement("div").Attr("data-chat-screen", "").Child(s.m.view)`.
- `Init(ctx dom.Ctx)`: start the heartbeat: `s.tick()` where
  `tick = func() { s.m.view.RefreshPeople(); s.m.checkUnread(); s.timer = time.AfterFunc(chatroom.HeartbeatIntervalSeconds*1000, s.tick) }`
  — the first call runs immediately (records presence at start). If `ctx != nil`, `ctx.OnCleanup(func(){ s.timer.Stop() })`
  (check `webtyp.com/time.Timer`'s method name; use whatever stops it).

### Stage 5 — `ui/groups.go`: the group admin screen

```go
func GroupsBrowser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error)
```
A `decktabs.DeckTabs` with two tabs (reference: `github.com/veltylabs/appointment_booking/ui/personal.go`):
1. **"Grupos"**: `crudview.New(crudview.Config{ParentID: GroupsID + ".list", Presenter: chatroom.NewGroupView(caller), IDs: ids})`
   returned as the panel (never discarded).
2. **"Miembros"**: struct `membersTab{ dom.Element; caller; tenantID; groupSel *dom.SignalString; groupOpts, boxes *dom.SignalNodes; msg *dom.SignalString; on []string; people []chatroom.Participant }`
   - `Init`: `OpListGroups` → options of a group `<select>` (`html.Option(id, name)`, first group selected);
     `OpListGroupCandidates` → `people`; then `loadMembers()`.
   - `<select>` `OnChange(e) { groupSel.Set(e.TargetValue()); loadMembers() }` — read the value from the event,
     never `GetID()`.
   - `loadMembers()`: `OpListGroupMembers` (`ListGroupMembersArgs{TenantId, RoomId}`) → `on` = their user ids;
     `boxes.Set(...)` **fresh**: one `<label>` per person with `<input type="checkbox">`
     `Attr("data-user-id", p.UserId)`, checked when in `on`, `OnChange(e) → add/remove p.UserId in on by e.TargetChecked()`.
   - "Guardar" → `OpSetGroupMembers` with `SetGroupMembersArgs{TenantId, RoomId, UserIds: []IdRef from on}`;
     message `"Miembros guardados."` or `err.Error()`.
   - No groups → `boxes.Set(Div().Text("Cree un grupo en la pestaña Grupos."))`.

### Stage 6 — `web/client.go`: runnable demo (`//go:build wasm`, package `main`)

Copy the chassis part of `github.com/veltylabs/patient_directory/web/client.go`. Then:
- `demoParticipants` implementing `chatroom.ParticipantReader`: `u1` "Ana (tú)", `u2` "Bruno", `u3` "Carla".
- `broker := &mock.Broker{}` is **both** `Deps.Publisher` and the `events.Subscriber` for `WithInbox`.
- `mod, err := chatroom.New(orm.New(mem.New()), chatroom.Deps{IDs: ids, Participants: demoParticipants{}, TenantID: demoTenantID, RetentionDays: 90, Publisher: broker, Clock: monotonic()})`
  where `monotonic()` returns a `func() int64` that answers `max(time.Now(), last+1)` (the seed writes several
  messages within the same browser millisecond; without it the demo's unread counts come out 0).
- `seed.Load(mod, demoTenantID, []string{"u1", "u2", "u3"})`
- `caller := loopback.ActingAs(demoTenantID, "u1", mod)`
- Modules: `ui.Browser(caller, ids, demoTenantID, ui.WithInbox(broker, "u1"))` and `ui.GroupsBrowser(caller, ids, demoTenantID)`.
- So the demo shows push working: a `time.AfterFunc(8000, …)` that sends, **as u2**, one message to the seeded
  direct room through the module directly (`mod.SendMessage(demoTenantID, "u2", data.DirectRoom.RoomId, "¿Tienes un momento?")`)
  — the module publishes on `InboxTopic("u1")` and the screen refreshes itself.
- `platformd.Platform{AppName: ui.DefaultLabel + " — demo", User: demoUser{}, Modules: [...], DefaultID: ui.ID}`, `dom.Append("body", p)`, `select {}`.

### Stage 7 — tests (`tests/`, package `tests`)

Stdlib build:
- `ui_source_test.go` is not possible from `tests/` (source is unexported) — instead test the mapping through
  the wasm tests below, and unit-test `clock`/`shortTime` by exporting nothing: put that test in
  `ui/time_internal_test.go` (package `ui`, the only in-package test allowed, because the helpers are unexported).
- `group_candidates_test.go` (Stage 1.5).

Wasm build — `tests/ui_live_wasm_test.go` (`//go:build wasm`). Helper `mountChat(t)`: real module over
`orm.New(mem.New())` with `Deps.Clock` = a **strictly increasing** fake clock (start at `time.Now()`, add
1 ms per call — the browser clock ties at millisecond resolution, and unread is `created_at > last_read_at`,
so equal timestamps make the counts flaky), the three demo participants, `mock.Broker` as publisher+subscriber, `seed.Load(... u1,u2,u3)`,
`loopback.ActingAs(tenant, "u1", mod)`, `ui.Browser(..., ui.WithInbox(broker, "u1"))`, `dom.Render(root, m.View())`
into a fresh root (copy the mount helper style of `github.com/veltylabs/room_layout/tests/ui_live_wasm_test.go`). Tests:
1. **Inbox lists u1's rooms**: 3 rows (General, Recepción, the direct room with Bruno). Query the rows by the
   selectors `inboxlist` renders (read `webtyp.com/components/inboxlist/inboxlist.go` for its row class /
   data attribute; do not invent one).
2. **Badge counts unread from others**: `m.(platformd.Badged).Badge().Count.Get()` is `"1"` and
   `Visible.Get()` is true. Why 1: sending marks the sender's own read point (`SendMessage` calls `MarkRead`),
   and u1 wrote last in the direct room, so only Bruno's General message ("Hola a todos.") is unread.
3. **Push refreshes**: `mod.SendMessage(tenant, "u2", directRoomID, "otro")` → badge count becomes `"2"`
   with no other call (the broker delivered `InboxTopic("u1")`).
4. **Heartbeat recorded presence**: after mount, `mod.ListParticipants(tenant, "u2")` shows u1 `Online: true`.
5. **Groups screen**: `ui.GroupsBrowser` mounted over `ActingAs(tenant, "u1", mod)`: the Miembros tab shows
   3 checkboxes `input[type='checkbox'][data-user-id]`, 2 of them checked (Recepción = u1, u2).

## 3. Documentation

- `docs/ARCHITECTURE.md` (missing today — create it, in Spanish like the code comments): scope; entities;
  the rules in `AGENTS.md` "Domain-specific notes"; ops table (all 13, with Access); push flow
  (`SendMessage` → `InboxTopic(recipient)` without payload → subscriber → `Refresh` via ops); heartbeat
  (`People` = `heartbeat`, every `HeartbeatIntervalSeconds`); composition-root example for an app
  (server: `chatroom.New(db, Deps{..., RetentionDays: 90, Publisher})`, `migrate.Migrate`, periodic
  `PurgeExpired`; browser: `ui.Browser(caller, ids, tenant, ui.WithInbox(sub, userID))`, `ui.GroupsBrowser(...)`).
- `docs/diagrams/database.md`: mermaid ERD of `chat_room`, `chat_member`, `chat_message`, `chat_presence`.
- `README.md`: quick start, ops table, "Demo: run `webtyp` at the repo root".

## 4. Acceptance criteria

```bash
gotest                                                                   # full suite green, wasm included
grep -rn "webtyp.com/ddl" --include=*.go . | grep -v "^./migrate/"       # → empty
grep -rn "MountOps\b\|migrate.Sync\|GetID()" --include=*.go .            # → empty
grep -rn '\.ID("' ui/ web/                                               # → empty
test -f docs/ARCHITECTURE.md && test -f docs/diagrams/database.md
```

## 5. Out of scope

System messages from other modules (v2), attachments, typing indicator, sound, patient chat.
Wiring into `mjosefa-cms` is a separate plan.
