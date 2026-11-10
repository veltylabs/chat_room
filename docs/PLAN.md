---
PLAN: "refactor!: chat_room on the crud-by-declaration framework — Allow and Rules in definition.go, persistence by crud, only domain rules in rule files"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 12282846242674632024
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Commit after every stage.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
> Never edit `definition_gen.go`: a build error there means `definition.go` or a rule file is wrong; fix that and run `definitionc` again.
> This plan is dispatched in one wave with the other 11 modules without external calls (D53), after `patient_directory` (the reference) was executed and approved by the owner.


# Plan: `chat_room` on the crud-by-declaration framework

Today 11 operations; 9 have no `Requires` (any signed-in person). With D24 those become grants to `rbac.RoleAuthenticated` that this module declares (`DefaultGrants`). `Member` has a composite key (`room_id/user_id`, D22). `RoomSummary`, `MessageView` and `MessageSentEvent` are not tables (the migration does not create them): remove their `DB:` marks. Reads that depend on who asks (my rooms, a room's messages) are read models whose function checks membership with `a.UserID`.

Reference implementation of the same migration (read it first, copy its shape): `patient_directory` — `definition.go` (`Definition`, `Allow`, `Rules`), `rule_patient_create.go`/`rule_patient_update.go`/`rule_patient_search.go` (the rules), `store.go`, `readers.go` (signatures kept, bodies on `m.store`).

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
go install webtyp.com/definitionc/cmd/definitionc@v0.1.41
go get webtyp.com/model@v0.2.18 webtyp.com/crud@v0.0.11 webtyp.com/orm@v0.12.30 webtyp.com/storage@v0.2.1 webtyp.com/ddl@v0.0.49 webtyp.com/router@v0.4.21 webtyp.com/definitionc@v0.1.41 webtyp.com/view@v0.6.72 webtyp.com/appkit@v0.2.33 webtyp.com/rpc@v0.2.21
```
Remove every `replace` from `go.mod`; `go mod tidy`. Before any change check, with `go doc`, that these exist: `model.ModuleDefinition`, `model.Rules` (`model.AddMyBusinessRule`, `model.HaveNoBusinessRule`), `crud.CurrentUser`, `crud.RuleAnswer`, `crud.Topic`, `crud.Loopback`, and that `definitionc` writes a `definition_gen.go` containing `func (m *Module) MountOperations(`. If one is missing, stop and say so in the PR.
Prerequisite of the whole wave: `patient_directory` was executed, reviewed and approved by the owner; it is the reference and the 17 other modules are dispatched together after it.

## Lessons of the reference module (`patient_directory`, executed locally)

- Needs `crud` ≥ v0.0.11 (a filter has no required field; v0.0.10 rejected every read of a table with a rule). `definitionc` ≥ v0.1.41 generates `Search(a, filter *X, text, m)`; a text of `""` is normal (the rule runs on every read).
- Running `definitionc` rewrites `go.mod` to older versions: run the `go get` list of «Tools» again after it, then `go mod tidy`; raise stale indirect dependencies (`rbac`, `mcp`, `appkit`) to the published ones.
- In tests, events reach the publisher given to `crud.LoopbackActingAs(pub, "u1", mod)`, not `Deps.Publisher`; a caller double takes `router.Operation`; a `ddl.Execer` double returns `(storage.Result, error)`; a rule that answers `0` fails with an internal error (assert the refusal and that nothing was stored).
- `langc sync .` can warn that other languages are untranslated: that is the repo's `lang.json`, not this plan.

## The rule of this plan (decisions D28–D50 of the architecture master)

The framework persists; the module declares and writes only domain rules.

| The module no longer writes | Who does it now |
|---|---|
| id, `tenant_id`, `created_at`/`updated_at`, the transaction loop, create/update/delete bodies, «not found», batch atomicity, idempotent retry by id, paging by `After`, `tenant_id` in every query, schema checks repeated from the model | `crud` + `orm.TenantStore` / `orm.GlobalStore` |
| an operation per verb (`list_*`, `save_*`, `upsert_*`, `delete_*`), its handler, `ops.go`, the permission resources and the event topics | `Allow` + `Rules` in `definition.go`; `definitionc` writes `MountOperations`; `crud` names the operation (`router.Operation{Module, Entity, Action}`; on the wire `GET\|POST\|PATCH\|DELETE /api/<module>/<entity>`, in MCP `<module>.<entity>.read\|create\|update\|delete`) and publishes `<module>.<entity>.created\|updated\|deleted` with a `*crud.Change` after the commit |
| an action with its own permission (approve, settle, receive stock…) | a **record created** in its own table, or a **read model** with its own name (D18) |

A module writes: `definition.go` (`var Definition = model.ModuleDefinition{…}` and, per table, `Allow` and `Rules`), one `rule_<entity>_<verb>.go` for every action declared `model.AddMyBusinessRule` (created once by `definitionc`; the skeleton does not compile until completed) and `store.go` (`func (m *Module) Store<Entity>() crud.Store`, one line per table). Every allowed action that has no rule below is declared `model.HaveNoBusinessRule`: `crud` alone checks it. Read the reference first: `patient_directory` (`definition.go`, `rule_patient_*.go`, `store.go`).

| Rule | Form (the receiver is the entity) |
|---|---|
| Create | `func (p *X) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error)` |
| Update | `func (next *X) Update(a crud.CurrentUser, tx crud.Tx, old *X, fields []string, m *Module) (crud.RuleAnswer, error)` (`next` carries only the sent `fields`) |
| Delete | `func (p *X) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error)` (`p` is the stored row) |
| Search (Read of a table) | `func (p *X) Search(a crud.CurrentUser, filter *X, text string, m *Module) ([]storage.Condition, error)` — AND conditions only; it runs on every read, `text` may be `""`, and `filter` is a record of the entity with only the fields the caller sent (D52: the entity is its own filter) |
| Read (model without a table) | `func (p *X) Read(a crud.CurrentUser, filter *X, args crud.ReadArgs, m *Module) (model.ModelSlice, error)` (`filter` carries the fields the caller sent, D52) |

A rule ends with `return crud.ApplyAndPublish, nil` (saves and announces) or `crud.ApplyWithoutPublishing` (saves without announcing); a rejection is `return 0, err`. It queries with `tx`, never with `m.db`. To write another row it calls `tx.Create`/`tx.UpdateFields`. The old `Committed` hook does not exist: `crud` publishes.

## Tables, stores and rules

An action with a rule below is declared `model.AddMyBusinessRule` in the table's `Rules`; every other allowed action is `model.HaveNoBusinessRule`. A `Search` below is the rule of `Read`.

| Table (`<X>Model`) | Store | `Allow` | Rules (`rule_<entity>_<verb>.go`) — where their rules come from today |
|---|---|---|---|
| `Room` | `orm.TenantStore` | `model.Create \| model.Read \| model.Update \| model.Delete` | groups only (admin): `Create`/`Update`: checks of `SaveGroup` (kind group); `Delete`: checks of `DeleteGroup`; direct rooms are never created through this table |
| `Member (key `room_id/user_id`)` | `orm.TenantStore` | `model.Create \| model.Read \| model.Update \| model.Delete` | `Create`/`Delete` (admin, group members): checks of `SetGroupMembers`; `Update` (anyone, own row only): fields `["last_read_id"]` and `old.UserId == a.UserID` (else field error) — mark read |
| `Message` | `orm.TenantStore` | `model.Create` | `Create` (anyone): checks of `SendMessage` — author = `a.UserID`, caller is a member of `room_id`, body ≤ `MaxBodyLength`; the `InboxTopic` event `SendMessage` publishes today is `crud`'s `chat_room.message.created` (body `*crud.Change`): `grep -rn "InboxTopic" /home/cesar/Dev/Project --include=*.go` and move each subscriber to the new topic and body, then delete `InboxTopic` |
| `Presence` | `orm.TenantStore` | `model.Create \| model.Update` | heartbeat (anyone): `Create` with `id = a.UserID`, `user_id = a.UserID`; `Update` only own row (`old.UserId == a.UserID`), fields `["last_seen"]`; the screen sends `_u` and, on «not found», `_c` once |
| `Direct (**new table** `chat_direct`)` | `orm.TenantStore` | `model.Create \| model.Read` | open a direct conversation (anyone): `Create` with `user_a`, `user_b` (composite key, D22: a pair has one row; wire id `model.JoinKey(user_a, user_b)`); the rule rejects with a field error unless `a.UserID` is one of them and `user_a < user_b`; it creates the direct `Room` and its two `Member` rows with `tx` (logic of `OpenDirect`) and stores `room_id`. Opening an existing pair again is the idempotent retry: crud returns the stored row |

## Read models (D18 — no table, `Allow: model.Read` only, `Rules.Read: model.AddMyBusinessRule`)

| Model | Filter | Answered by |
|---|---|---|
| `RoomSummary` (existing; no `DB:` marks) | none (the caller) | `ListRooms(tenant, a.UserID)` |
| `MessageView` (existing; no `DB:` marks) | `MessagesFilter{room_id, after_id string}` | `ListMessages(tenant, a.UserID, …)` — not a member → `model.ErrorNotFound` |
| `Participant` (existing) | none | `ListParticipants(tenant, a.UserID)`; `list_group_candidates` reads it too |

A read model is a `model.Definition` **without** `DB:` marks, with `Allow: model.Read` and `Rules: model.Rules{Read: model.AddMyBusinessRule}`, listed in `Definition.Entities`; `definitionc` mounts it and creates its `rule_<model name>_read.go`. Its permission resource is its own `ModelName`: list it in the README permissions table (roles must be granted it).

## Operation mapping (every operation of today has exactly one row)

«After» is `router.Operation{Module, Entity, Action}` written `<entity>.<verb>`; the wire form is `crud`'s (REST `/api/<module>/<entity>` by method, MCP `<module>.<entity>.<verb>`).

| Today | After |
|---|---|
| `list_participants / list_group_candidates` | read model `Participant` |
| `heartbeat` | `chat_presence.update` (or `_c` the first time) |
| `open_direct` | `chat_direct.create` with `{user_a, user_b}` (sorted; caller is one) |
| `list_rooms` | read model `RoomSummary` |
| `list_messages` | read model `MessageView` |
| `send_message` | `chat_message.create` |
| `mark_read` | `chat_member.update` (id `room/user`) with `["last_read_id"]` |
| `list_groups (+save/delete)` | `chat_room.read` (filter `{kind:"group"}`) / `_c` / `_u` / `_d` |
| `set_group_members` | `chat_member.create` / `_d` |
| `list_group_members` | `chat_member.read` with filter `{room_id}` |

## Stage 1 — `definition.go`

- Set `Allow` on each table exactly as in «Tables, stores and rules»; a table not listed there gets no `Allow` (internal).
- Mark `ServerOnly: true`: `tenant_id`, `created_at`, `updated_at`, and every field the server fills (actor/author fields).
- Delete every `*Args` / `*Result` definition that only carried a removed operation
  (`grep -n "ArgsModel\|ResultModel" model.go` → only the ones this plan keeps).
- New table `DirectModel` (name `chat_direct`): `user_a` and `user_b` (both `DB: &model.FieldDB{PK: true}`; the client sends both, sorted, one of them being itself), `tenant_id`, `room_id` (`ServerOnly`), `created_at`; add it to `migrate/`. The screen lists directs whole (`view.Page{First: view.All}`).
- Remove the `DB:` marks of `RoomSummaryModel`, `MessageViewModel`, `MessageSentEventModel`.
- New exported `var DefaultGrants = []model.Grant{{Resource: model.ResourceOf(&MessageModel), Actions: model.Create}, {Resource: model.ResourceOf(&PresenceModel), Actions: model.Create | model.Update}, {Resource: model.ResourceOf(&DirectModel), Actions: model.Create | model.Read}, {Resource: model.ResourceOf(&MemberModel), Actions: model.Update}, {Resource: "room_summary", Actions: model.Read}, {Resource: "message_view", Actions: model.Read}, {Resource: "participant", Actions: model.Read}}` (use the read models' `ModelName()` constants, not literals); `module/server.go` sets `ModuleServer.DefaultGrants: chatroom.DefaultGrants` (D24, applied by appkit to `rbac.RoleAuthenticated`).
- **Keep every `*Args` model that an exported reader's signature still uses** (`grep -n "^func (m \*Module) [A-Z][A-Za-z]*([^)]*Args" *.go` lists them): mark each `Wire: true`; delete only the `*Args` no remaining signature uses.
- D52 (the entity is its own filter): a filter model (`*Filter`, `*FilterModel`, an `*ArgsModel` that only carried filter values) is not kept: its fields become fields of the read model or table (`OmitEmpty`), and the rule reads them from its receiver; delete the filter type and its `Wire`/`NewFilter` plumbing.
- Add `var Definition = model.ModuleDefinition{Name: "chat_room", Entities: []model.Definition{…every table and read model of this plan…}}` and, on every table and read model, `Rules` as «Tables, stores and rules» says (`AddMyBusinessRule` where a rule is named, `HaveNoBusinessRule` for the other allowed actions; a read model: `Read: model.AddMyBusinessRule`). Delete the hand topics (`Topic*`) and `Err*`/resources no code uses any more.

## Stage 2 — `module.go`

`Deps` keeps its fields; add `IDs model.IDGenerator` (required, `depErr("IDs")` when nil) if it is missing.
`Module` gains `store *orm.Store`, built once in `New`: `m.store = orm.TenantStore(db, deps.TenantID, deps.IDs)`.
Delete the hand-written persistence (create/update/upsert/patch/delete/list/get bodies that only move rows) once its rules moved to the rule files: the row-moving bodies of `SaveGroup`, `DeleteGroup`, `SetGroupMembers`, `SendMessage`, `MarkRead`, `Heartbeat`, `OpenDirect` (their checks move to the rules); every handler without `Requires`.

## Stage 3 — generate, then write the rules

1. Delete `ops.go`, `crud.go`, `queries.go`, `model_crud.go`, `model_orm.go` and `definition_crud.go` (those that exist). Run `definitionc`: it writes one `definition_gen.go` (the types, the schemas and, on `Module`, `ModelName()`, `Definition()`, `MountOperations()`) and the skeleton `rule_<entity>_<verb>.go` of every action declared `AddMyBusinessRule`.
2. Add `store.go`: for each table of «Tables, stores and rules» that has a Store, `// Store<Entity> is the store of <entity>.` `func (m *Module) Store<Entity>() crud.Store { return m.store }` (the module's global store for a table the plan lists with `orm.GlobalStore`).
3. Fill each skeleton with the body of the rule named in «Tables, stores and rules» (and «Read models»), in the forms of «The rule of this plan». An old rule `func(c crud.Caller, tx, rec) error` becomes the rule method with the same body, `c.UserID` becomes `a.UserID`, a nil error becomes `return crud.ApplyAndPublish, nil`, an error `e` becomes `return 0, e`.
4. Specific to this module: Every rule named above is a rule file. `markRead` rejects any field but `last_read_id`. `Deps.RetentionDays` and `Deps.Clock` keep their use in the rules that need them.
Delete every `Op*` constant, `res*`/`Resource*` variable, every `op*` handler, every `view.MountSave`/`view.MountDelete`/`reg.Operation` call and the `webtyp.com/router`, `webtyp.com/rpc` imports that only they used (`grep -rn "reg.Operation\|view.MountSave\|view.MountDelete\|router.Context\|rpc.WriteError" --include=*.go . | grep -v _test` → empty).

## Stage 4 — readers used by other modules

Signatures of every exported reader stay (other modules and apps call them). Bodies read through `m.store` (`ReadOne`, `ReadAll` with `crud.Query`), never `m.db` directly; a `tenantID` parameter different from the module's tenant answers «not found» (`found=false` / empty, `err=nil`). Keep `Heartbeat`, `ListMessages`, `SendMessage`, `MarkRead`, `ListParticipants`, `OpenDirect`, `ListRooms`, `ListGroups`, `SaveGroup`, `SetGroupMembers`, `ListGroupCandidates`, `ListGroupMembers` (writers run the same rules inside `m.store.Atomic`).

## Stage 5 — browser side (D15, D16)

- `view.go` (`//go:build browser`): every `view.NewCallerLister(caller, view.Ops{…}, newList)` takes
  `func() model.TableSlice { return &<Entity>List{} }` and `view.Ops` has no `Allow` (the screen reads `<Entity>List.Allow()`).
- `module/browser.go`, `module/server.go` (if present): `appkit.ModuleBrowser`/`appkit.BrowserDeps` → `view.ModuleBrowser`/`view.BrowserDeps`.
- Demo: `seed/demo.go` (`//go:build wasm`) returning `view.Demo{Label, Timezone: "America/Santiago", ActingAs: "u1",
  Modules: []crud.Module{mod}, Migrations: <the module's ddl.Step list, or nil if it has no migrations>,
  Screens: func(d view.BrowserDeps) []view.Screen { return ui.Screens(<what today's demo main passes>) }}`, built with
  `orm.New(mem.New())`, `New(db, Deps{…})` with the same fakes today's demo uses, and `seed.Load` (when the module has one).
  Delete the old demo: `web/browser.go` (`test ! -e web`).
- `grep -rn "webtyp.com/layout\|appkit.BrowserDeps\|appkit.ModuleBrowser\|router/loopback" --include=*.go . | grep -v "^./ui/"` → empty. `ui/` keeps its `webtyp.com/layout/chatview` and `crudview` import until phase L6 of the architecture master (it builds that piece of the chassis); do not touch it here.

## Stage 6 — tests (`tests/`, through `crud.LoopbackActingAs(nil, "u1", mod)`; rows in `docs/TESTS.md`)

Delete tests of removed code. Operations are `router.Operation{Module: ModelName, Entity: "<entity>", Action: model.<Action>}`, never a string; fixtures follow `crud` v0.0.10 (`reg.AddStoredEntity`, `crud.CurrentUser`). Add `TestMount_MatchesDefinition` (`crud.Mount(reg, nil, nil, mod)` returns nil) and one test that an allowed write publishes a `*crud.Change` on `crud.Topic(ModelName, "<entity>", model.<Action>)`. Required:

| Test | Setup → call → expected |
|---|---|
| `TestMessage_NonMemberRejected` | u3 not in the room sends → error; nothing stored |
| `TestMessages_NonMemberCannotRead` | read model `MessageView` for a room u3 is not in → `model.ErrorNotFound` |
| `TestMember_MarkReadOnlyOwnRow` | u1 updates `room/u2` → field error |
| `TestDirect_SamePairOnce` | u1 opens `{u1,u2}` twice, then u2 opens `{u1,u2}` → one room |
| `TestDirect_CallerMustBeInPair` | u3 opens `{u1,u2}` → field error |
| `TestPresence_OnlyOwn` | u1 updates u2's presence → field error |
| `TestDefaultGrants_SignedInCanSend` | with `rbac.EnsureGrants(…, RoleAuthenticated, DefaultGrants)`, a user with no roles can create a message (through `rpc.Routes` + rbac `Can`) |
| `TestDemo_Valid` (`//go:build wasm`) | `seed.Demo()` → no error, `Validate()` nil, at least one screen |

## Stage 7 — translations and documentation (Spanish, veltylabs)

`langc sync .` (zero pending, zero reuse warnings). `README.md` (≤150 lines, 4 mandatory sections; «Sin/Con»: hand-written
operations vs `Allow` + rules) with the operations table of the mapping above. `docs/ARCHITECTURE.md`: rules vs persistence.
Translate to Spanish every README, `docs/*`, diagram and code comment still in English; exported symbols without a usage example get one.

## Delivery check

`gotest` passes; `langc sync .` clean; every grep of this plan empty. A tree that does not compile or whose tests did not run is a
failed delivery.
