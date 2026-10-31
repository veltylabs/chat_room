package chatroom

import (
	"webtyp.com/input"
	"webtyp.com/model"
	"webtyp.com/time"
)

const ModelName = "chat_room"

// Tipos de sala — únicos valores de chat_room.kind.
const (
	KindDirect    = "direct"
	KindGroup     = "group"
	KindBroadcast = "broadcast"
)

const (
	BroadcastName                      = "General"
	MaxBodyLength                      = 2000
	PreviewLength                      = 80
	HeartbeatIntervalSeconds           = 30 // lo usa la vista
	presenceWindow           time.Unix = 90
	DefaultMessagesPageLimit           = 50
)

var RoomModel = model.Definition{
	Name: "chat_room",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "kind", Type: model.Text(), NotNull: true},
		{Name: "name", Type: input.Text(), Item: model.ItemLabel, OmitEmpty: true, Permitted: model.Permitted{Maximum: 60}}, // solo group y broadcast; lo edita el admin en el formulario de grupos
		{Name: "direct_key", Type: model.Text(), OmitEmpty: true},                                                           // solo direct: "<idMenor>|<idMayor>"
		{Name: "created_at", Type: model.NanoTime(), NotNull: true},
		{Name: "updated_at", Type: model.NanoTime(), OmitEmpty: true},
	},
}

var MemberModel = model.Definition{
	Name: "chat_member",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "room_id", Type: model.Text(), Ref: &RoomModel, DB: &model.FieldDB{PK: true, RefColumn: "id"}, NotNull: true},
		{Name: "user_id", Type: model.Text(), DB: &model.FieldDB{PK: true}, NotNull: true},
		{Name: "last_read_at", Type: model.NanoTime()},
	},
}

var MessageModel = model.Definition{
	Name: "chat_message",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "room_id", Type: model.Text(), Ref: &RoomModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "sender_id", Type: model.Text(), NotNull: true},
		{Name: "sender_label", Type: model.Text(), NotNull: true}, // copia del nombre al enviar
		{Name: "body", Type: model.Text(), NotNull: true},
		{Name: "created_at", Type: model.NanoTime(), NotNull: true}, // nanosegundos
	},
}

var PresenceModel = model.Definition{
	Name: "chat_presence",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "user_id", Type: model.Text(), DB: &model.FieldDB{PK: true}, NotNull: true},
		{Name: "last_seen", Type: model.NanoTime(), NotNull: true},
	},
}

var IdRefModel = model.Definition{
	Name: "id_ref",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), NotNull: true},
	},
}

// Estructuras de argumentos (Structs for arguments)

var OpenDirectArgsModel = model.Definition{
	Name: "open_direct_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "other_user_id", Type: model.Text(), NotNull: true},
	},
}

var ListRoomsArgsModel = model.Definition{
	Name: "list_rooms_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
	},
}

var ListMessagesArgsModel = model.Definition{
	Name: "list_messages_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "room_id", Type: model.Text(), Ref: &RoomModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "after_id", Type: model.Text()},
		{Name: "limit", Type: model.Int()},
	},
}

var SendMessageArgsModel = model.Definition{
	Name: "send_message_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "room_id", Type: model.Text(), Ref: &RoomModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "body", Type: input.Textarea(), NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: MaxBodyLength}},
	},
}

var MarkReadArgsModel = model.Definition{
	Name: "mark_read_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "room_id", Type: model.Text(), Ref: &RoomModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
	},
}

var HeartbeatArgsModel = model.Definition{
	Name: "heartbeat_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
	},
}

var ListGroupsArgsModel = model.Definition{
	Name: "list_groups_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
	},
}

var SetGroupMembersArgsModel = model.Definition{
	Name: "set_group_members_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "room_id", Type: model.Text(), Ref: &RoomModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "user_ids", Type: model.StructSlice(&IdRefModel)},
	},
}

var ListGroupMembersArgsModel = model.Definition{
	Name: "list_group_members_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "room_id", Type: model.Text(), Ref: &RoomModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
	},
}

// Definiciones de salida / resultado (Output / Result Definitions)

var ParticipantModel = model.Definition{
	Name: "participant",
	Fields: model.Fields{
		{Name: "user_id", Type: model.Text(), NotNull: true},
		{Name: "label", Type: model.Text(), NotNull: true},
		{Name: "online", Type: model.Bool(), NotNull: true},
	},
}

var RoomSummaryModel = model.Definition{
	Name: "room_summary",
	Fields: model.Fields{
		{Name: "room_id", Type: model.Text(), Ref: &RoomModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "kind", Type: model.Text(), NotNull: true},
		{Name: "title", Type: model.Text(), NotNull: true},
		{Name: "unread", Type: model.Int(), NotNull: true},
		{Name: "last_message_at", Type: model.NanoTime(), NotNull: true},
		{Name: "last_preview", Type: model.Text(), NotNull: true},
	},
}

var MessageViewModel = model.Definition{
	Name: "message_view",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), NotNull: true},
		{Name: "room_id", Type: model.Text(), Ref: &RoomModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "sender_id", Type: model.Text(), NotNull: true},
		{Name: "sender_label", Type: model.Text(), NotNull: true},
		{Name: "body", Type: model.Text(), NotNull: true},
		{Name: "created_at", Type: model.NanoTime(), NotNull: true},
		{Name: "mine", Type: model.Bool(), NotNull: true},
		{Name: "read", Type: model.Bool(), NotNull: true},
	},
}

var MessageSentEventModel = model.Definition{
	Name: "message_sent_event",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "room_id", Type: model.Text(), Ref: &RoomModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "message_id", Type: model.Text(), NotNull: true},
		{Name: "sender_id", Type: model.Text(), NotNull: true},
	},
}

// Errores (texto exacto) (Errors (exact text))
const (
	ErrNotFound        model.NotFound = "chat_room: not found"
	ErrTenantRequired  model.Invalid  = "chat_room: tenant_id is required"
	ErrUnauthenticated model.Invalid  = "chat_room: authenticated user required"
	ErrNotMember       model.Invalid  = "chat_room: not a member of this room"
	ErrUnknownUser     model.Invalid  = "chat_room: unknown participant"
	ErrSelfDirect      model.Invalid  = "chat_room: cannot open a direct room with yourself"
	ErrEmptyBody       model.Invalid  = "chat_room: message body is empty"
	ErrBodyTooLong     model.Invalid  = "chat_room: message body is too long"
	ErrNotGroup        model.Invalid  = "chat_room: room is not a group"
	ErrCannotRead      model.Invalid  = "chat_room: cannot read the request"
)

// Eventos y Tópicos (Events & Topics)
const (
	TopicMessageSent = "chat_room.message.sent" // payload MessageSentEvent: tenant_id, room_id, message_id, sender_id — NUNCA el body
	TopicGroupSaved  = "chat_room.group.saved"  // payload: la fila chat_room
)

const inboxTopicPrefix = "chat_room.inbox."

// InboxTopic es el canal de avisos de un usuario. La app suscribe la conexión push de cada
// usuario autenticado SOLO a su propio InboxTopic. El evento que se publica ahí no lleva payload:
// significa "algo cambió en tus salas, vuelve a pedir".
func InboxTopic(userID string) string {
	return inboxTopicPrefix + userID
}
