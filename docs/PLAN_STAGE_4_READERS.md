← [Stage 3](PLAN_STAGE_3_RULES.md) | Next → [Stage 5](PLAN_STAGE_5_BROWSER.md)

# Stage 4 — readers and writers on `m.store`; delete the old paths

Goal: no `m.db` is left in `group.go`, `message.go`, `room.go`, `presence.go`; `ops.go` is deleted. The **signatures of every exported method stay** (other modules and apps call them): `ListGroups`, `SaveGroup`, `SetGroupMembers`, `ListGroupCandidates`, `ListGroupMembers`, `DeleteGroup`, `PurgeExpired`, `ListMessages`, `SendMessage`, `MarkRead`, `ListParticipants`, `OpenDirect`, `ListRooms`, `Heartbeat`. Translate with the table of Stage 2.4. Edit **one method at a time, by hand**, then `gotest`.

Rules for every method:
- `tenantID == ""` means `m.tenantID`. A `tenantID` different from `m.tenantID` answers «not found»: a list returns an empty slice and a nil error; a single lookup or a writer returns `ErrNotFound`.
- Readers use `m.store` as the `crud.Tx` (`firstRow(m.store, …)`, `allRows(m.store, …)`, `m.store.ReadOne`). Writers use `m.store.Atomic(func(tx crud.Tx) error {...})`.
- A method that has a rule calls the SAME check function the rule calls (`checkSendMessage`, `deleteGroupRows`, `validGroupName`, `isRoomMember`): one rule, one place.
- No `map` (this package compiles to WASM): use slices and loops, as the old code did.
- Constants `newRoom`, `newMember`, `newMessage`, `newPresence` are not needed: write the constructor inline, `func() model.Model { return &Member{} }`.

## 4.1 New internal helpers (in `room.go`, replacing the old ones)

```go
// ensureBroadcast creates the tenant's general room the first time anyone needs it.
func (m *Module) ensureBroadcast() error {
	if _, found, err := firstRow(m.store, func() model.Model { return &Room{} }, storage.Eq(Room_.Kind, KindBroadcast)); err != nil || found {
		return err
	}
	return m.store.Atomic(func(tx crud.Tx) error {
		if _, found, err := firstRow(tx, func() model.Model { return &Room{} }, storage.Eq(Room_.Kind, KindBroadcast)); err != nil || found {
			return err
		}
		return tx.Create(&Room{Id: m.ids.NewID(), Kind: KindBroadcast, Name: BroadcastName, CreatedAt: m.nowNano()})
	})
}
```
Every old call `m.ensureBroadcast(tenantID)` becomes `m.ensureBroadcast()`. `isMember` is deleted (use `isRoomMember(tx, userID, room)` of `checks.go`). `getPresences` becomes `allRows(m.store, func() model.Model { return &Presence{} }, nil)` converted to `[]Presence`.

## 4.2 Worked example — `ListMessages` (the model for every reader)

```go
func (m *Module) ListMessages(tenantID, userID, roomID, afterID string, limit int) ([]MessageView, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	if tenantID == "" {
		tenantID = m.tenantID
	}
	if tenantID != m.tenantID {
		return nil, ErrNotFound
	}
	if limit <= 0 {
		limit = DefaultMessagesPageLimit
	}
	if err := m.ensureBroadcast(); err != nil {
		return nil, err
	}
	room, err := roomOf(m.store, roomID)
	if err != nil {
		return nil, err
	}
	isMember, err := m.isRoomMember(m.store, userID, room)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotFound // a person outside the room must not learn it exists
	}
	conditions := []storage.Condition{storage.Eq(Message_.RoomId, roomID)}
	order := []storage.Order{storage.Desc(Message_.CreatedAt)}
	var after Message
	if afterID != "" {
		if err := m.store.ReadOne(&after, afterID); err != nil || after.RoomId != roomID {
			return nil, ErrNotFound
		}
		conditions = append(conditions, storage.Gte(Message_.CreatedAt, int64(after.CreatedAt)))
		order = []storage.Order{storage.Asc(Message_.CreatedAt)}
	}
	var messages []Message
	err = m.store.ReadAll(func() model.Model { return &Message{} },
		crud.Query{Conditions: conditions, Order: order, Limit: int64(limit) + 1},
		func(r model.Model) { messages = append(messages, *r.(*Message)) })
	if err != nil {
		return nil, err
	}
	if afterID != "" {
		var newer []Message
		for _, message := range messages {
			if message.CreatedAt > after.CreatedAt || (message.CreatedAt == after.CreatedAt && message.Id > after.Id) {
				newer = append(newer, message)
			}
		}
		messages = newer
		if len(messages) > limit {
			messages = messages[:limit]
		}
	} else {
		if len(messages) > limit {
			messages = messages[:limit]
		}
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 { // newest-first read, oldest-first answer
			messages[i], messages[j] = messages[j], messages[i]
		}
	}
	var otherReadAt time.Nano
	if room.Kind == KindDirect {
		row, found, err := firstRow(m.store, func() model.Model { return &Member{} },
			storage.Eq(Member_.RoomId, roomID), storage.Neq(Member_.UserId, userID))
		if err != nil {
			return nil, err
		}
		if found {
			otherReadAt = row.(*Member).LastReadAt
		}
	}
	views := make([]MessageView, 0, len(messages))
	for _, message := range messages {
		mine := message.SenderId == userID
		views = append(views, MessageView{
			Id: message.Id, RoomId: message.RoomId, SenderId: message.SenderId, SenderLabel: message.SenderLabel,
			Body: message.Body, CreatedAt: message.CreatedAt, Mine: mine,
			Read: room.Kind == KindDirect && mine && otherReadAt >= message.CreatedAt,
		})
	}
	return views, nil
}
```
`ListMessages` answers `ErrNotFound` (not `ErrNotMember`) to a person outside the room: the read model returns it unchanged (`model.ErrorNotFound`). `SendMessage` keeps `ErrNotMember`.

## 4.3 Writers — exact bodies

**`SendMessage`** (keeps its publishing exactly as today; the check is shared with the rule)
```go
func (m *Module) SendMessage(tenantID, userID, roomID, body string) (MessageView, error) {
	if userID == "" {
		return MessageView{}, ErrUnauthenticated
	}
	if tenantID == "" {
		tenantID = m.tenantID
	}
	if tenantID != m.tenantID {
		return MessageView{}, ErrNotFound
	}
	if err := m.ensureBroadcast(); err != nil {
		return MessageView{}, err
	}
	message := Message{RoomId: roomID, Body: body}
	var room Room
	err := m.store.Atomic(func(tx crud.Tx) error {
		var checkErr error
		room, checkErr = m.checkSendMessage(tx, userID, &message)
		if checkErr != nil {
			return checkErr
		}
		message.Id = m.ids.NewID()
		return tx.Create(&message)
	})
	if err != nil {
		return MessageView{}, err
	}
	_ = m.MarkRead(tenantID, userID, roomID)
	recipients, recipientsErr := m.recipientIDs(room, userID)
	if recipientsErr == nil && m.publisher != nil {
		m.publisher.Publish(events.Event{Topic: TopicMessageSent, Payload: &MessageSentEvent{TenantId: tenantID, RoomId: roomID, MessageId: message.Id, SenderId: userID}})
		for _, recipient := range recipients {
			m.publisher.Publish(events.Event{Topic: InboxTopic(recipient)})
		}
	}
	return MessageView{Id: message.Id, RoomId: message.RoomId, SenderId: message.SenderId, SenderLabel: message.SenderLabel,
		Body: message.Body, CreatedAt: message.CreatedAt, Mine: true}, nil
}
```
(`Read` of a direct room for the author is computed by the next `ListMessages`; the old code filled it from the other member's read time: keep that lookup exactly as in 4.2 if a test needs `Read` in the answer.) `getRoomRecipientIDs` becomes `recipientIDs(room Room, senderID string) ([]string, error)`: same logic, the broadcast branch uses `m.participants.ParticipantOptions(m.tenantID)`, the other branch `allRows(m.store, …Member…, nil, storage.Eq(Member_.RoomId, room.Id))`.

**`MarkRead`**
```go
func (m *Module) MarkRead(tenantID, userID, roomID string) error {
	if userID == "" {
		return ErrUnauthenticated
	}
	if tenantID == "" {
		tenantID = m.tenantID
	}
	if tenantID != m.tenantID {
		return ErrNotFound
	}
	if err := m.ensureBroadcast(); err != nil {
		return err
	}
	return m.store.Atomic(func(tx crud.Tx) error {
		room, err := roomOf(tx, roomID)
		if err != nil {
			return err
		}
		isMember, err := m.isRoomMember(tx, userID, room)
		if err != nil {
			return err
		}
		if !isMember {
			return ErrNotMember
		}
		now := m.nowNano()
		row, found, err := firstRow(tx, func() model.Model { return &Member{} },
			storage.Eq(Member_.RoomId, roomID), storage.Eq(Member_.UserId, userID))
		if err != nil {
			return err
		}
		if found {
			member := row.(*Member)
			member.LastReadAt = now
			return tx.UpdateFields(member, model.JoinKey(member.RoomId, member.UserId), []string{Member_.LastReadAt})
		}
		return tx.Create(&Member{RoomId: roomID, UserId: userID, LastReadAt: now})
	})
}
```
**`Heartbeat`**: check `userID`, tenant, `isParticipant` (`ErrUnknownUser` when not), then
```go
	now := m.nowNano()
	err := m.store.Atomic(func(tx crud.Tx) error {
		row, found, err := firstRow(tx, func() model.Model { return &Presence{} }, storage.Eq(Presence_.UserId, userID))
		if err != nil {
			return err
		}
		if found {
			presence := row.(*Presence)
			presence.LastSeen = now
			return tx.UpdateFields(presence, presence.UserId, []string{Presence_.LastSeen})
		}
		return tx.Create(&Presence{UserId: userID, LastSeen: now})
	})
	if err != nil {
		return nil, err
	}
	return m.ListParticipants(tenantID, userID)
```
**`SaveGroup`**: `name, err := validGroupName(name)` (a bad name is now a field error, not `ErrNotFound`); then in `m.store.Atomic`: with `id != ""`: `room, err := roomOf(tx, id)`, `room.Kind != KindGroup → ErrNotGroup`, `room.Name = name; room.UpdatedAt = now`, `tx.UpdateFields(&room, room.Id, []string{Room_.Name, Room_.UpdatedAt})`, `saved = room`; with `id == ""`: `saved = Room{Id: m.ids.NewID(), Kind: KindGroup, Name: name, CreatedAt: now}; tx.Create(&saved)`. After the commit publish `TopicGroupSaved` with `&saved` exactly as today.
**`DeleteGroup`**: `m.store.Atomic`: `room, err := roomOf(tx, id)`; `room.Kind != KindGroup → ErrNotGroup`; `deleteGroupRows(tx, id)`; `tx.Delete(&room, id)`.
**`SetGroupMembers`** (the kept «sync»; it does NOT delete and recreate every row: it adds the missing and removes the extra, so `last_read_at` of staying members is untouched)
```go
	var added, removed []string
	err := m.store.Atomic(func(tx crud.Tx) error {
		room, err := roomOf(tx, roomID)
		if err != nil {
			return err
		}
		if room.Kind != KindGroup {
			return ErrNotGroup
		}
		for _, userID := range userIDs {
			if _, ok, err := m.isParticipant(userID); err != nil {
				return err
			} else if !ok {
				return ErrUnknownUser
			}
		}
		rows, err := allRows(tx, func() model.Model { return &Member{} }, nil, storage.Eq(Member_.RoomId, roomID))
		if err != nil {
			return err
		}
		for _, row := range rows {
			member := row.(*Member)
			if !containsString(userIDs, member.UserId) {
				if err := tx.Delete(row, model.JoinKey(member.RoomId, member.UserId)); err != nil {
					return err
				}
				removed = append(removed, member.UserId)
			}
		}
		for _, userID := range userIDs {
			present := false
			for _, row := range rows {
				if row.(*Member).UserId == userID {
					present = true
				}
			}
			if !present {
				if err := tx.Create(&Member{RoomId: roomID, UserId: userID}); err != nil {
					return err
				}
				added = append(added, userID)
			}
		}
		return nil
	})
```
then the same `InboxTopic` publishing for `append(added, removed...)` as today. Add `func containsString(list []string, value string) bool` to `checks.go`.
**`OpenDirect`**: the validations as today (`ErrUnauthenticated`, `ErrSelfDirect`, `ErrUnknownUser` through `isParticipant(otherUserID)`, whose label is the title); then in `m.store.Atomic`: `firstRow` of `Room` with `storage.Eq(Room_.Kind, KindDirect)` and `storage.Eq(Room_.DirectKey, directKey)`; if found use it; if not, create the room (`Room{Id: m.ids.NewID(), Kind: KindDirect, DirectKey: directKey, CreatedAt: now}`), its two `Member` rows and the pair row `Direct{UserA: low, UserB: high, RoomId: room.Id, CreatedAt: now}` (low/high = the two ids sorted). Return `m.getRoomSummary(room, userID, title)`.
**`PurgeExpired`**: `cutoff` as today; loop: `rows, err := allRows(m.store, …Message…, nil, storage.Lt(Message_.CreatedAt, int64(cutoff)))`; stop when empty; `m.store.Atomic` deleting each (`tx.Delete(row, row.(*Message).Id)`), `count++`.
**Pure readers** (`ListGroups`, `ListGroupCandidates`, `ListGroupMembers`, `ListParticipants`, `ListRooms`, `getRoomSummary`): translate mechanically with the Stage 2.4 table; the logic of building `Participant`, `RoomSummary` stays byte for byte. `ListGroups(search, after, limit)` uses `crud.Query{Conditions: …, After: after, Limit: limit}` (the `Room` key is the single column `id`, so `After` is valid and `Order` stays empty).

## 4.4 Delete

`ops.go` (and every `Op*` constant, `res*` variable, `op*` handler it holds), `crud.go`, `queries.go` if they exist, and every import that only they used (`webtyp.com/router`, `webtyp.com/rpc`). Verify: `grep -rn "reg.Operation\|view.MountSave\|view.MountDelete\|router.Context\|rpc.WriteError" --include=*.go . | grep -v _test` → empty; `grep -n "m\.db\." group.go message.go room.go presence.go` → empty.

`gotest` compiles now; some old tests may fail because of the two changed answers (`ListMessages` → `ErrNotFound`, bad group name → field error): fix those assertions in Stage 6, not the code. Commit and push.
