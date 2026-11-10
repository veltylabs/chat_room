← [Index](PLAN.md) | Next → [Stage 2](PLAN_STAGE_2_MODULE_STORE.md)

# Stage 1 — tools and `definition.go`

## 1.1 Tools (run first, exactly)

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
go install webtyp.com/definitionc/cmd/definitionc@v0.1.42
export PATH="$PATH:$(go env GOPATH)/bin"
go get webtyp.com/model@v0.2.19 webtyp.com/crud@v0.0.12 webtyp.com/orm@v0.12.33 webtyp.com/storage@v0.2.1 webtyp.com/ddl@v0.0.49 webtyp.com/router@v0.4.22 webtyp.com/definitionc@v0.1.42 webtyp.com/view@v0.6.74 webtyp.com/appkit@v0.2.35
go mod tidy
```

Environment: `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, `GOFLAGS=-mod=mod`. Remove every `replace` from `go.mod`. **`definitionc` rewrites `go.mod` to older versions every time it runs: after EVERY run repeat the `go get` line above and `go mod tidy`.** If a module cannot be fetched, stop and say so in the PR; never invent a hash.

Facts of the framework you need (they were asked before; do not ask again):

- Operation arguments are `crud.CreateArgs{Records []model.Model}`, `crud.UpdateArgs{IDs []string, Fields []string, Record model.Model}`, `crud.DeleteArgs{IDs []string}`, `crud.ReadArgs{ID string, Search string, After string, Limit int64, Filter model.Model}`; never a bare entity pointer, never a generated `*List`. `crud.Change` has only `Record model.Model` and `Fields []string`.
- `model.Pointer`, `model.List` and `model.Map` do not exist; the kinds are flat (`model.Text`, `Int`, `Bool`, `NanoTime`, `Float`, `Blob`, `Raw`, `IntSlice`, `input.*`).
- Inside a rule `tx` is a `crud.Tx`: `tx.ReadOne(&rec, id)` (error of kind `model.ErrorNotFound` when absent), `tx.ReadAll(newRow func() model.Model, crud.Query{Conditions []storage.Condition, Order []storage.Order, Limit int64}, onRow func(model.Model))`, `tx.Create(&rec)`, `tx.UpdateFields(&rec, id, []string{"col"})`, `tx.Delete(&rec, id)`. Conditions are `storage.Eq/Neq/Gt/Gte/Lt/Lte/In/Like(column, value)` (AND only); the tenant is already applied by the store: **never add a `tenant_id` condition**. A value given to `Gt/Lt/Gte/Lte` is converted with `int64(...)` when it is a `time.Nano`.

## 1.2 `definition.go` — exact edits

Work on the existing file; keep every constant and the three `Kind*` groups.

1. **Delete** the eight `*ArgsModel` definitions (`OpenDirectArgsModel`, `ListRoomsArgsModel`, `ListMessagesArgsModel`, `SendMessageArgsModel`, `MarkReadArgsModel`, `HeartbeatArgsModel`, `SetGroupMembersArgsModel`, `ListGroupMembersArgsModel`) and `IdRefModel`. Verify no exported method uses one: `grep -n "ArgsModel\|IdRefModel" *.go | grep -v "^definition"` must be empty after Stage 4 (it can still show `ops.go` until then).
2. **Tables.** Set `Allow` and `Rules` on each; add `Wire`-free `ServerOnly: true` where listed:

| Model | `Allow` | `Rules` | `ServerOnly: true` on |
|---|---|---|---|
| `RoomModel` | `model.Create \| model.Read \| model.Update \| model.Delete` | all four `model.AddMyBusinessRule` | `tenant_id`, `direct_key`, `created_at`, `updated_at` |
| `MemberModel` | `model.Create \| model.Read \| model.Update \| model.Delete` | all four `model.AddMyBusinessRule` | `tenant_id` |
| `MessageModel` | `model.Create` | `Create: model.AddMyBusinessRule` | `tenant_id`, `sender_id`, `sender_label`, `created_at` |
| `PresenceModel` | `model.Create \| model.Update` | both `model.AddMyBusinessRule` | `tenant_id` |
| `DirectModel` (new) | `model.Create \| model.Read` | `Create: model.AddMyBusinessRule`, `Read: model.HaveNoBusinessRule` | `tenant_id`, `room_id`, `created_at` |

   `last_read_at` (member) and `last_seen` (presence) are NOT `ServerOnly`: the client names them in `Fields` and the rule overwrites the value with the clock.
3. **New table `DirectModel`**, written exactly:

```go
// DirectModel is one row per pair of people who chat directly; its rule creates the direct room and its two members.
var DirectModel = model.Definition{
	Name:  "chat_direct",
	Allow: model.Create | model.Read,
	Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule},
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text(), NotNull: true, ServerOnly: true},
		{Name: "user_a", Type: model.Text(), DB: &model.FieldDB{PK: true}, NotNull: true},
		{Name: "user_b", Type: model.Text(), DB: &model.FieldDB{PK: true}, NotNull: true},
		{Name: "room_id", Type: model.Text(), NotNull: true, ServerOnly: true},
		{Name: "created_at", Type: model.NanoTime(), NotNull: true, ServerOnly: true},
	},
}
```
4. **Read models** (`ParticipantModel`, `RoomSummaryModel`, `MessageViewModel`): add `Wire: true`, `Allow: model.Read`, `Rules: model.Rules{Read: model.AddMyBusinessRule}`; remove every `Ref` and `DB:` from their fields (`room_id` keeps only `Type: model.Text()`). Add to `MessageViewModel` the filter field `{Name: "after_id", Type: model.Text(), OmitEmpty: true}` and mark its existing `room_id` `OmitEmpty: true`. `MessageSentEventModel`: add `Wire: true`, remove the `Ref`/`DB:` of `room_id`; it is NOT in `Entities` and has no `Allow`.
5. **Error constants to add** next to the others (exact text):

```go
	ErrNameInvalid       model.Invalid = "chat_room: group name must have 1 to 60 characters"
	ErrFieldNotEditable  model.Invalid = "chat_room: that field cannot be changed"
	ErrRoomRequired      model.Invalid = "chat_room: room_id is required"
	ErrDirectPairInvalid model.Invalid = "chat_room: a direct pair needs user_a smaller than user_b"
```
   Delete nothing else (`TopicMessageSent`, `TopicGroupSaved`, `InboxTopic` stay: other repos subscribe to them).
6. **`DefaultGrants`** (new, exported; `module/server.go` uses it in Stage 5), written exactly:

```go
// DefaultGrants are the permissions every signed-in person gets for chatting (applied by appkit to the authenticated role).
//
// Example: ModuleServer{DefaultGrants: chatroom.DefaultGrants}
var DefaultGrants = []model.Grant{
	{Resource: model.ResourceOf(&MessageModel), Actions: model.Create},
	{Resource: model.ResourceOf(&PresenceModel), Actions: model.Create | model.Update},
	{Resource: model.ResourceOf(&DirectModel), Actions: model.Create | model.Read},
	{Resource: model.ResourceOf(&MemberModel), Actions: model.Update},
	{Resource: model.Resource(RoomSummaryModel.Name), Actions: model.Read},
	{Resource: model.Resource(MessageViewModel.Name), Actions: model.Read},
	{Resource: model.Resource(ParticipantModel.Name), Actions: model.Read},
}
```
7. **`Definition`** (new), after all models:

```go
var Definition = model.ModuleDefinition{
	Name: "chat_room",
	Entities: []model.Definition{
		RoomModel, MemberModel, MessageModel, PresenceModel, DirectModel,
		ParticipantModel, RoomSummaryModel, MessageViewModel,
	},
}
```
8. `migrate/migrate.go`: the `Sync(...)` list becomes `&chatroom.Room{}, &chatroom.Member{}, &chatroom.Message{}, &chatroom.Presence{}, &chatroom.Direct{}` (the read models are not tables).

## 1.3 Done when

`go vet ./...` may still fail (`ops.go` and the old methods reference removed types): that is expected until Stage 4. Run `definitionc` ONLY in Stage 3. Commit and push this stage.
