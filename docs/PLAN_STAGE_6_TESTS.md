← [Stage 5](PLAN_STAGE_5_BROWSER.md) | Next → [Stage 7](PLAN_STAGE_7_DOCS.md)

# Stage 6 — tests

## 6.1 What already exists stays

The tests of the kept methods (`direct_test.go`, `group_test.go`, `inbox_test.go`, `message_test.go`, `presence_test.go`, `unread_test.go`, `retention_test.go`, `sentinels_test.go`, `new_test.go`, `broadcast_test.go`, `group_candidates_test.go`, `coverage_extra_test.go`, `extra_branches_test.go`, `screen_save_test.go`) keep testing the kept methods. Fix only what the two behaviour changes of Stage 4 break: a non-member reading messages gets `ErrNotFound`; a bad group name is a field error (`model.KindOf(err) == model.ErrorInvalid`), not `ErrNotFound`. A test of a removed `op*` handler or of `ops.go` is deleted; do not delete any other test. `ui_live_browser_test.go` (`browser` tag) must compile against the new `router.Operation` signature of `Call`/`Dispatch` (`Call(op router.Operation, args model.Encodable, into model.Decodable, done func(error))`).

## 6.2 New crud-level tests, `tests/crud_test.go` (package `tests`), written with these helpers

```go
package tests

import (
	"testing"

	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/crud"
	"webtyp.com/events"
	"webtyp.com/model"
	"webtyp.com/router"
)

// call runs one operation of the module as `user` through the same loopback the demo uses and answers the callback error.
func call(user string, pub events.Publisher, mod *chatroom.Module, entity string, action model.Action, args model.Encodable, into model.Decodable) error {
	var callErr error
	crud.LoopbackActingAs(pub, user, mod).Call(
		router.Operation{Module: chatroom.ModelName, Entity: entity, Action: action},
		args, into, func(err error) { callErr = err })
	return callErr
}

// failedField answers the field a refusal points at, or "".
func failedField(err error) string {
	if f, ok := err.(interface{ FailedField() string }); ok {
		return f.FailedField()
	}
	return ""
}
```
Use the module the existing `tests/setup_test.go` builds (read it: it already has the participants double with `u1`, `u2`, `u3`, a clock and an id generator); if its constructor returns more values, adapt only the first line of each test. A room of the broadcast kind exists after the first `ListRooms`/`SendMessage`; create a group the test needs through `m.SaveGroup("", "", "Team")` and its members through `m.SetGroupMembers("", group.Id, []string{"u1", "u2"})`. **Every created record carries an `Id` where the entity has one** (`'id required'` otherwise); `Member` and `Presence` have composite or user keys, so their `Create` records carry `RoomId`+`UserId` / `UserId`.

| Test (all in `tests/crud_test.go`) | setup → call → expected |
|---|---|
| `TestMessage_NonMemberRejected` | group `Team` with `u1`,`u2`; as `u3`: `call("u3", nil, m, "chat_message", model.Create, &crud.CreateArgs{Records: []model.Model{&chatroom.Message{RoomId: group.Id, Body: "hi"}}}, nil)` → error; reading messages as `u1` shows none |
| `TestMessage_CreateSetsSenderFields` | as `u1` the same create → no error; reading as `u1` through the read model (below) shows `SenderId == "u1"` and `Mine == true` |
| `TestMessages_NonMemberCannotRead` | `var list chatroom.MessageViewList`; `call("u3", nil, m, "message_view", model.Read, &crud.ReadArgs{Filter: &chatroom.MessageView{RoomId: group.Id}}, &list)` → `model.KindOf(err) == model.ErrorNotFound` |
| `TestMessages_ReadModelNeedsRoom` | `call("u1", …, "message_view", model.Read, &crud.ReadArgs{Filter: &chatroom.MessageView{}}, &list)` → `failedField(err) == "room_id"` |
| `TestMember_MarkReadOnlyOwnRow` | as `u1`: `call("u1", nil, m, "chat_member", model.Update, &crud.UpdateArgs{IDs: []string{model.JoinKey(group.Id, "u2")}, Fields: []string{"last_read_at"}, Record: &chatroom.Member{LastReadAt: 1}}, nil)` → `failedField(err) == "user_id"`; the same on `model.JoinKey(group.Id, "u1")` → no error and the stored `LastReadAt` is greater than 1 |
| `TestMember_UpdateOnlyLastRead` | as `u1` on its own row with `Fields: []string{"room_id"}` → `failedField(err) == "room_id"` |
| `TestMember_CreateNeedsKnownGroupUser` | `chat_member.create` `{RoomId: group.Id, UserId: "nobody"}` → `failedField(err) == "user_id"`; with `UserId: "u3"` → no error; the same create again → `model.KindOf(err) == model.ErrorConflict` |
| `TestDirect_SamePairOnce` | as `u1`: `chat_direct.create` `{UserA: "u1", UserB: "u2"}` → no error; again → `model.ErrorConflict`; the kept `m.OpenDirect("", "u2", "u1")` answers the same `RoomId` as the first |
| `TestDirect_CallerMustBeInPair` | as `u3`: `chat_direct.create` `{UserA: "u1", UserB: "u2"}` → `failedField(err) == "user_a"` |
| `TestDirect_PairMustBeSorted` | as `u2`: `{UserA: "u2", UserB: "u1"}` → `failedField(err) == "user_a"` |
| `TestPresence_OnlyOwn` | as `u1`: `chat_presence.update` on id `"u2"` with `Fields: []string{"last_seen"}` → `failedField(err) == "user_id"` |
| `TestGroup_DeleteCascades` | group with two members and 3 messages; `chat_room.delete` as `u1` (`&crud.DeleteArgs{IDs: []string{group.Id}}`) → no error; `chat_member` reads with filter `{RoomId: group.Id}` are empty and `m.ListMessages` for that room answers `ErrNotFound` |
| `TestGroup_DeleteRejectsNonGroup` | `chat_room.delete` on the broadcast room → `ErrNotGroup` text in the error |
| `TestMount_MatchesDefinition` | `crud.Mount(nil, nil, nil, m)` returns nil |
| `TestCreate_PublishesChange` | a `chat_room.create` of a group with a publisher: one event on `crud.Topic(chatroom.ModelName, (&chatroom.Room{}).ModelName(), model.Create)` whose payload is `*crud.Change` |
| `TestDefaultGrants_SignedInCanSend` | `chatroom.DefaultGrants` contains a grant with `Resource == model.ResourceOf(&chatroom.MessageModel)` and `Actions&model.Create != 0`, and none with `model.Delete` for any resource (a pure check of the slice: no rbac stack) |
| `TestDemo_Valid` (`tests/demo_test.go`, `//go:build wasm`) | copy https://github.com/veltylabs/patient_directory/blob/main/tests/demo_test.go replacing the import path by `github.com/veltylabs/chat_room/seed` |

A rule that refuses is observed through the callback error; assert the field with `failedField` and the kind with `model.KindOf`. Events reach the publisher given as the FIRST argument of `crud.LoopbackActingAs(pub, user, mod)`, never `Deps.Publisher`.

## 6.3 `docs/TESTS.md`

One row per `func Test…` of the repo (old and new), under the header `| Test | Rule |` and separator `|---|---|`, each row `| [TestName](../tests/file_test.go) | one line rule |` — the link IS the file; **no File column** (it fails the `gotest` tests-index check). Rules in Spanish.

`gotest` is green on every destination and «tests index» is ✅. Commit and push.
