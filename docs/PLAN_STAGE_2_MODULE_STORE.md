← [Stage 1](PLAN_STAGE_1_TOOLS_DEFINITION.md) | Next → [Stage 3](PLAN_STAGE_3_RULES.md)

# Stage 2 — `Module.store`, `store.go`, read helpers

## 2.1 `module.go`

In `type Module struct` add `store *orm.Store` after `db`. In `New`, in the returned literal add `store: orm.TenantStore(db, deps.TenantID, deps.IDs),`. Nothing else changes (`Deps`, `ModelName()` stay).

## 2.2 `store.go` (new), written exactly

```go
package chatroom

import "webtyp.com/crud"

// StoreRoom is the store of the rooms.
func (m *Module) StoreRoom() crud.Store { return m.store }

// StoreMember is the store of the room members.
func (m *Module) StoreMember() crud.Store { return m.store }

// StoreMessage is the store of the messages.
func (m *Module) StoreMessage() crud.Store { return m.store }

// StorePresence is the store of the presence rows.
func (m *Module) StorePresence() crud.Store { return m.store }

// StoreDirect is the store of the direct pairs.
func (m *Module) StoreDirect() crud.Store { return m.store }
```

## 2.3 `rows.go` (new): the two helpers every rule and reader uses, written exactly

```go
package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

// firstRow answers the first row that matches conds, and whether there is one.
// It reads through tx (a rule) or through the store (a reader): both are a crud.Tx.
//
// Example: row, found, err := firstRow(tx, func() model.Model { return &Member{} }, storage.Eq(Member_.RoomId, id))
func firstRow(tx crud.Tx, newRow func() model.Model, conds ...storage.Condition) (model.Model, bool, error) {
	var first model.Model
	err := tx.ReadAll(newRow, crud.Query{Conditions: conds, Limit: 1}, func(r model.Model) {
		if first == nil {
			first = r
		}
	})
	return first, first != nil, err
}

// allRows answers every row that matches conds, in the order asked (at most crud.MaxPage rows).
//
// Example: rows, err := allRows(tx, func() model.Model { return &Member{} }, nil, storage.Eq(Member_.RoomId, id))
func allRows(tx crud.Tx, newRow func() model.Model, order []storage.Order, conds ...storage.Condition) ([]model.Model, error) {
	var rows []model.Model
	err := tx.ReadAll(newRow, crud.Query{Conditions: conds, Order: order, Limit: crud.MaxPage}, func(r model.Model) {
		rows = append(rows, r)
	})
	return rows, err
}
```

`Member_`, `Room_` and the other `<Entity>_` constants come from the generated `definition_gen.go` (Stage 3). Until then this file does not compile: write it now, commit it with Stage 3.

## 2.4 Translation table for Stages 3 and 4 (old `m.db` call → new)

| Old | New |
|---|---|
| `m.db.Query(&room).Where("tenant_id").Eq(t).Where("id").Eq(id).ReadOne()` then `orm.IsNotFound(err)` | `err := tx.ReadOne(&room, id)` then `model.KindOf(err) == model.ErrorNotFound` |
| `m.db.Query(&mem).Where(...tenant...).Where("room_id").Eq(r).Where("user_id").Eq(u).ReadOne()` | `row, found, err := firstRow(tx, func() model.Model { return &Member{} }, storage.Eq(Member_.RoomId, r), storage.Eq(Member_.UserId, u))` then `mem := row.(*Member)` |
| `m.db.Query(&Member{}).Where(...tenant...).Where("room_id").Eq(r).ReadAll(newRow, cb)` | `rows, err := allRows(tx, func() model.Model { return &Member{} }, nil, storage.Eq(Member_.RoomId, r))` then `rows[i].(*Member)` |
| `.OrderBy("created_at").Asc()` | `[]storage.Order{storage.Asc(Message_.CreatedAt)}` as the `order` argument |
| `.Where("kind").Eq(KindGroup)` | `storage.Eq(Room_.Kind, KindGroup)` |
| `.Where("name").Like("%"+s+"%")` | `storage.Like(Room_.Name, "%"+s+"%")` |
| `.Where("created_at").Lt(int64(cutoff))` | `storage.Lt(Message_.CreatedAt, int64(cutoff))` |
| `m.db.Create(&x)` | `tx.Create(&x)` (id, `tenant_id`, `created_at`, `updated_at` are stamped by `crud`/the store when empty) |
| `m.db.UpdateFields(&x, cols, orm.Eq(...tenant...), orm.Eq(Pk, v))` | `tx.UpdateFields(&x, id, cols)` |
| `m.db.Delete(&x, orm.Eq(...))` | `tx.Delete(&x, id)` |
| `m.db.Tx(func(tx *orm.DB) error {...})` | `m.store.Atomic(func(tx crud.Tx) error {...})` |

A table with a composite key (`Member`: `room_id/user_id`) has the wire id `model.JoinKey(room_id, user_id)`; `tx.UpdateFields(&mem, model.JoinKey(mem.RoomId, mem.UserId), cols)` and `tx.Delete(&mem, model.JoinKey(...))`.

Commit and push.
