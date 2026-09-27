package seed

import (
	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/fmt"
)

type Data struct {
	Group           chatroom.Room
	GroupMembers    []chatroom.Participant
	DirectRoom      chatroom.RoomSummary
	DirectMessages  []chatroom.MessageView
	GeneralMessages []chatroom.MessageView
}

func Load(m *chatroom.Module, tenantID string, userIDs []string) (Data, error) {
	if len(userIDs) < 2 {
		return Data{}, fmt.Err("seed: at least 2 user IDs required")
	}

	user1 := userIDs[0]
	user2 := userIDs[1]

	// 1. Create group "Recepción" and set members
	group, err := m.SaveGroup(tenantID, "", "Recepción")
	if err != nil {
		return Data{}, err
	}

	err = m.SetGroupMembers(tenantID, group.Id, []string{user1, user2})
	if err != nil {
		return Data{}, err
	}

	groupMembers, err := m.ListGroupMembers(tenantID, group.Id)
	if err != nil {
		return Data{}, err
	}

	// 2. Direct room between user1 and user2 with 3 messages
	directRoom, err := m.OpenDirect(tenantID, user1, user2)
	if err != nil {
		return Data{}, err
	}

	m1, err := m.SendMessage(tenantID, user1, directRoom.RoomId, "Hola, ¿cómo estás?")
	if err != nil {
		return Data{}, err
	}

	m2, err := m.SendMessage(tenantID, user2, directRoom.RoomId, "Todo bien, ¿y tú?")
	if err != nil {
		return Data{}, err
	}

	m3, err := m.SendMessage(tenantID, user1, directRoom.RoomId, "Bien también, gracias.")
	if err != nil {
		return Data{}, err
	}

	// 3. General (broadcast) room with 2 messages
	rooms, err := m.ListRooms(tenantID, user1)
	if err != nil {
		return Data{}, err
	}

	var broadcastRoomID string
	for _, r := range rooms {
		if r.Kind == chatroom.KindBroadcast {
			broadcastRoomID = r.RoomId
			break
		}
	}

	gen1, err := m.SendMessage(tenantID, user1, broadcastRoomID, "Bienvenidos al chat General.")
	if err != nil {
		return Data{}, err
	}

	gen2, err := m.SendMessage(tenantID, user2, broadcastRoomID, "Hola a todos.")
	if err != nil {
		return Data{}, err
	}

	return Data{
		Group:           group,
		GroupMembers:    groupMembers,
		DirectRoom:      directRoom,
		DirectMessages:  []chatroom.MessageView{m1, m2, m3},
		GeneralMessages: []chatroom.MessageView{gen1, gen2},
	}, nil
}
