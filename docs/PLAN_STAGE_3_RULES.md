← [Stage 2](PLAN_STAGE_2_MODULE_STORE.md) | Next → [Stage 4](PLAN_STAGE_4_READERS.md)

# Stage 3 — generate, then write the rules

## 3.1 Generate

1. Delete `model_orm.go` and `definition_crud.go` if they exist (not `ops.go` yet: Stage 4 deletes it). Run `definitionc` in the repo root. It writes `definition_gen.go` (types `Room`, `Member`, `Message`, `Presence`, `Direct`, `Participant`, `RoomSummary`, `MessageView`; their `<Entity>List`; the constants `Room_`, `Member_`…; `ModelName()`, `Definition()`, `MountOperations()`) and one skeleton per action declared `AddMyBusinessRule`. Then repeat the `go get` line of Stage 1 and `go mod tidy`.
2. Expected skeleton files (if a name differs, keep the generated one): `rule_chat_room_{create,read,update,delete}.go`, `rule_chat_member_{create,read,update,delete}.go`, `rule_chat_message_create.go`, `rule_chat_presence_{create,update}.go`, `rule_chat_direct_create.go`, `rule_participant_read.go`, `rule_room_summary_read.go`, `rule_message_view_read.go`. A table `Read` declared `AddMyBusinessRule` is a `Search` method; a read model's `Read` is a `Read` method. **Keep the signatures definitionc wrote**; only replace the body.
3. `rows.go` and `store.go` (Stage 2) compile now.

## 3.2 `checks.go` (new): the checks both a rule and a kept writer method use, written exactly

```go
package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

// validGroupName trims a group name and checks it has 1 to 60 characters.
//
// Example: name, err := validGroupName("  Nurses ")
func validGroupName(name string) (string, error) {
	trimmed := trimSpaces(name)
	length := len([]rune(trimmed))
	if length < 1 || length > 60 {
		return "", &model.FieldError{Field: Room_.Name, Err: ErrNameInvalid}
	}
	return trimmed, nil
}

// isParticipant answers the visible name of a person who can chat, and whether they are one.
func (m *Module) isParticipant(userID string) (string, bool, error) {
	options, err := m.participants.ParticipantOptions(m.tenantID)
	if err != nil {
		return "", false, err
	}
	for _, option := range options {
		if option.Key == userID {
			return option.Value, true, nil
		}
	}
	return "", false, nil
}

// roomOf reads a room through tx; a missing room is ErrNotFound.
func roomOf(tx crud.Tx, id string) (Room, error) {
	var room Room
	if err := tx.ReadOne(&room, id); err != nil {
		if model.KindOf(err) == model.ErrorNotFound {
			return Room{}, ErrNotFound
		}
		return Room{}, err
	}
	return room, nil
}

// isRoomMember: everyone who can chat is in the broadcast room; any other room needs a Member row.
func (m *Module) isRoomMember(tx crud.Tx, userID string, room Room) (bool, error) {
	if room.Kind == KindBroadcast {
		_, ok, err := m.isParticipant(userID)
		return ok, err
	}
	_, found, err := firstRow(tx, func() model.Model { return &Member{} },
		storage.Eq(Member_.RoomId, room.Id), storage.Eq(Member_.UserId, userID))
	return found, err
}

// checkSendMessage validates a message of userID and fills the fields the server owns. It is the single place of
// the rules of sending: the rule of Message.Create and the kept method SendMessage both call it.
func (m *Module) checkSendMessage(tx crud.Tx, userID string, msg *Message) (Room, error) {
	if userID == "" {
		return Room{}, ErrUnauthenticated
	}
	room, err := roomOf(tx, msg.RoomId)
	if err != nil {
		return Room{}, err
	}
	isMember, err := m.isRoomMember(tx, userID, room)
	if err != nil {
		return Room{}, err
	}
	if !isMember {
		return Room{}, ErrNotMember
	}
	body := trimSpaces(msg.Body)
	length := len([]rune(body))
	if length == 0 {
		return Room{}, ErrEmptyBody
	}
	if length > MaxBodyLength {
		return Room{}, ErrBodyTooLong
	}
	label, _, err := m.isParticipant(userID)
	if err != nil {
		return Room{}, err
	}
	if label == "" {
		label = userID
	}
	msg.SenderId = userID
	msg.SenderLabel = label
	msg.Body = body
	msg.CreatedAt = m.nowNano()
	return room, nil
}

// deleteGroupRows deletes the messages and the members of a group room (not the room row itself) through tx. It is the
// single cascade: the rule of Room.Delete and the kept method DeleteGroup both call it. Ids are collected before
// deleting, and messages go in rounds because one read answers at most crud.MaxPage rows.
func deleteGroupRows(tx crud.Tx, roomID string) error {
	for {
		rows, err := allRows(tx, func() model.Model { return &Message{} }, nil, storage.Eq(Message_.RoomId, roomID))
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			if err := tx.Delete(row, row.(*Message).Id); err != nil {
				return err
			}
		}
	}
	members, err := allRows(tx, func() model.Model { return &Member{} }, nil, storage.Eq(Member_.RoomId, roomID))
	if err != nil {
		return err
	}
	for _, row := range members {
		member := row.(*Member)
		if err := tx.Delete(row, model.JoinKey(member.RoomId, member.UserId)); err != nil {
			return err
		}
	}
	return nil
}

// onlyFields refuses a field that is not in allowed.
func onlyFields(fields []string, allowed ...string) error {
	for _, field := range fields {
		permitted := false
		for _, name := range allowed {
			if field == name {
				permitted = true
			}
		}
		if !permitted {
			return &model.FieldError{Field: field, Err: ErrFieldNotEditable}
		}
	}
	return nil
}
```

## 3.3 The rule files (replace each generated body; imports as the file needs)

A rule that accepts ends `return crud.ApplyAndPublish, nil`; one that refuses ends `return 0, err`. In an `Update` rule the record `next` is written by `crud` AFTER the rule, with exactly the sent `fields`: assigning `next.Field = value` inside the rule changes what is stored.

**`rule_chat_room_create.go`** (groups only; direct and broadcast rooms are never created through this table)
```go
func (p *Room) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	name, err := validGroupName(p.Name)
	if err != nil {
		return 0, err
	}
	p.Kind = KindGroup
	p.Name = name
	p.DirectKey = ""
	return crud.ApplyAndPublish, nil
}
```
**`rule_chat_room_update.go`**
```go
func (next *Room) Update(a crud.CurrentUser, tx crud.Tx, old *Room, fields []string, m *Module) (crud.RuleAnswer, error) {
	if old.Kind != KindGroup {
		return 0, ErrNotGroup
	}
	if err := onlyFields(fields, Room_.Name); err != nil {
		return 0, err
	}
	name, err := validGroupName(next.Name)
	if err != nil {
		return 0, err
	}
	next.Name = name
	return crud.ApplyAndPublish, nil
}
```
**`rule_chat_room_delete.go`** (the whole cascade runs in `tx`, so it commits or rolls back as one)
```go
func (p *Room) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if p.Kind != KindGroup {
		return 0, ErrNotGroup
	}
	if err := deleteGroupRows(tx, p.Id); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
```
**`rule_chat_room_read.go`** (`Search`: filter by kind and by name)
```go
func (p *Room) Search(a crud.CurrentUser, filter *Room, text string, m *Module) ([]storage.Condition, error) {
	var conditions []storage.Condition
	if filter != nil && filter.Kind != "" {
		conditions = append(conditions, storage.Eq(Room_.Kind, filter.Kind))
	}
	if text != "" {
		conditions = append(conditions, storage.Like(Room_.Name, "%"+text+"%"))
	}
	return conditions, nil
}
```
**`rule_chat_member_create.go`** — ONE member per operation (a batch is a `CreateArgs` with several records, run in one transaction). There is no «sync members» operation: the kept method `SetGroupMembers` (Stage 4) computes the difference itself. A duplicate `room_id/user_id` is answered by `crud` as a conflict (composite primary key), not by this rule.
```go
func (p *Member) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	room, err := roomOf(tx, p.RoomId)
	if err != nil {
		return 0, err
	}
	if room.Kind != KindGroup {
		return 0, ErrNotGroup
	}
	if _, ok, err := m.isParticipant(p.UserId); err != nil {
		return 0, err
	} else if !ok {
		return 0, &model.FieldError{Field: Member_.UserId, Err: ErrUnknownUser}
	}
	return crud.ApplyAndPublish, nil
}
```
**`rule_chat_member_delete.go`**
```go
func (p *Member) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	room, err := roomOf(tx, p.RoomId)
	if err != nil {
		return 0, err
	}
	if room.Kind != KindGroup {
		return 0, ErrNotGroup
	}
	return crud.ApplyAndPublish, nil
}
```
**`rule_chat_member_update.go`** (mark read: anyone, own row only; the clock decides the value)
```go
func (next *Member) Update(a crud.CurrentUser, tx crud.Tx, old *Member, fields []string, m *Module) (crud.RuleAnswer, error) {
	if old.UserId != a.UserID {
		return 0, &model.FieldError{Field: Member_.UserId, Err: ErrNotMember}
	}
	if err := onlyFields(fields, Member_.LastReadAt); err != nil {
		return 0, err
	}
	next.LastReadAt = m.nowNano()
	return crud.ApplyAndPublish, nil
}
```
**`rule_chat_member_read.go`**
```go
func (p *Member) Search(a crud.CurrentUser, filter *Member, text string, m *Module) ([]storage.Condition, error) {
	var conditions []storage.Condition
	if filter != nil && filter.RoomId != "" {
		conditions = append(conditions, storage.Eq(Member_.RoomId, filter.RoomId))
	}
	return conditions, nil
}
```
**`rule_chat_message_create.go`**
```go
func (p *Message) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if _, err := m.checkSendMessage(tx, a.UserID, p); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
```
**`rule_chat_presence_create.go`** (heartbeat, first time)
```go
func (p *Presence) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if p.UserId != a.UserID {
		return 0, &model.FieldError{Field: Presence_.UserId, Err: ErrUnauthenticated}
	}
	if _, ok, err := m.isParticipant(a.UserID); err != nil {
		return 0, err
	} else if !ok {
		return 0, &model.FieldError{Field: Presence_.UserId, Err: ErrUnknownUser}
	}
	p.LastSeen = m.nowNano()
	return crud.ApplyAndPublish, nil
}
```
**`rule_chat_presence_update.go`**
```go
func (next *Presence) Update(a crud.CurrentUser, tx crud.Tx, old *Presence, fields []string, m *Module) (crud.RuleAnswer, error) {
	if old.UserId != a.UserID {
		return 0, &model.FieldError{Field: Presence_.UserId, Err: ErrUnauthenticated}
	}
	if err := onlyFields(fields, Presence_.LastSeen); err != nil {
		return 0, err
	}
	next.LastSeen = m.nowNano()
	return crud.ApplyAndPublish, nil
}
```
**`rule_chat_direct_create.go`** (a pair has one row; the rule creates the direct room and its two members in `tx`; opening the same pair again is answered by `crud` as a conflict on the composite key, and the kept method `OpenDirect` is the idempotent form)
```go
func (p *Direct) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if p.UserA >= p.UserB {
		return 0, &model.FieldError{Field: Direct_.UserA, Err: ErrDirectPairInvalid}
	}
	if a.UserID != p.UserA && a.UserID != p.UserB {
		return 0, &model.FieldError{Field: Direct_.UserA, Err: ErrNotMember}
	}
	other := p.UserA
	if a.UserID == p.UserA {
		other = p.UserB
	}
	if _, ok, err := m.isParticipant(other); err != nil {
		return 0, err
	} else if !ok {
		return 0, &model.FieldError{Field: Direct_.UserB, Err: ErrUnknownUser}
	}
	now := m.nowNano()
	room := Room{Id: m.ids.NewID(), Kind: KindDirect, DirectKey: makeDirectKey(p.UserA, p.UserB), CreatedAt: now}
	if err := tx.Create(&room); err != nil {
		return 0, err
	}
	for _, user := range []string{p.UserA, p.UserB} {
		if err := tx.Create(&Member{RoomId: room.Id, UserId: user}); err != nil {
			return 0, err
		}
	}
	p.RoomId = room.Id
	p.CreatedAt = now
	return crud.ApplyAndPublish, nil
}
```
**Read models** (the signatures are the generated ones, with `filter`). They call the kept readers of Stage 4; each wraps the rows in the generated list type and returns a pointer to it:

```go
// rule_participant_read.go
func (p *Participant) Read(a crud.CurrentUser, filter *Participant, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	rows, err := m.ListParticipants(m.tenantID, a.UserID)
	if err != nil {
		return nil, err
	}
	list := ParticipantList{}
	for i := range rows {
		row := rows[i]
		list = append(list, &row)
	}
	return &list, nil
}
```
`rule_room_summary_read.go`: the same shape with `m.ListRooms(m.tenantID, a.UserID)` and `RoomSummaryList`.
`rule_message_view_read.go`: the same shape with `RoomId` required:
```go
func (p *MessageView) Read(a crud.CurrentUser, filter *MessageView, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	if filter == nil || filter.RoomId == "" {
		return nil, &model.FieldError{Field: MessageView_.RoomId, Err: ErrRoomRequired}
	}
	rows, err := m.ListMessages(m.tenantID, a.UserID, filter.RoomId, filter.AfterId, int(args.Limit))
	if err != nil {
		return nil, err
	}
	list := MessageViewList{}
	for i := range rows {
		row := rows[i]
		list = append(list, &row)
	}
	return &list, nil
}
```
(`filter.AfterId` is the generated name of the `after_id` field; if `definitionc` names it differently, use the generated name.) A person who is not a member of the room gets `ErrNotFound` from `ListMessages` (Stage 4 makes it so): the rule returns that error unchanged.

## 3.4 Done when

`go vet ./...` compiles the rule files (the module may still fail on `ops.go` and the old methods until Stage 4). Commit, push, and **open the pull request now** (title: the `PLAN:` line).
