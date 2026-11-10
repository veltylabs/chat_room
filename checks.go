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

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
