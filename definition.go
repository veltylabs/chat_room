package chatroom

import (
	"webtyp.com/input"
	"webtyp.com/model"
	"webtyp.com/time"
)

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
	Name:  "chat_room",
	Allow: model.Create | model.Read | model.Update | model.Delete,
	Rules: model.Rules{
		Create: model.AddMyBusinessRule,
		Read:   model.AddMyBusinessRule,
		Update: model.AddMyBusinessRule,
		Delete: model.AddMyBusinessRule,
	},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true, ServerOnly: true},
		{Name: "kind", Type: model.Text(), NotNull: true},
		{Name: "name", Type: input.Text(), Item: model.ItemLabel, OmitEmpty: true, Permitted: model.Permitted{Maximum: 60}}, // solo group y broadcast; lo edita el admin en el formulario de grupos
		{Name: "direct_key", Type: model.Text(), OmitEmpty: true, ServerOnly: true},                                       // solo direct: "<idMenor>|<idMayor>"
		{Name: "created_at", Type: model.NanoTime(), NotNull: true, ServerOnly: true},
		{Name: "updated_at", Type: model.NanoTime(), OmitEmpty: true, ServerOnly: true},
	},
}

var MemberModel = model.Definition{
	Name:  "chat_member",
	Allow: model.Create | model.Read | model.Update | model.Delete,
	Rules: model.Rules{
		Create: model.AddMyBusinessRule,
		Read:   model.AddMyBusinessRule,
		Update: model.AddMyBusinessRule,
		Delete: model.AddMyBusinessRule,
	},
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text(), NotNull: true, ServerOnly: true},
		{Name: "room_id", Type: model.Text(), Ref: &RoomModel, DB: &model.FieldDB{PK: true, RefColumn: "id"}, NotNull: true},
		{Name: "user_id", Type: model.Text(), DB: &model.FieldDB{PK: true}, NotNull: true},
		{Name: "last_read_at", Type: model.NanoTime()},
	},
}

var MessageModel = model.Definition{
	Name:  "chat_message",
	Allow: model.Create,
	Rules: model.Rules{Create: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true, ServerOnly: true},
		{Name: "room_id", Type: model.Text(), Ref: &RoomModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "sender_id", Type: model.Text(), NotNull: true, ServerOnly: true},
		{Name: "sender_label", Type: model.Text(), NotNull: true, ServerOnly: true}, // copia del nombre al enviar
		{Name: "body", Type: model.Text(), NotNull: true},
		{Name: "created_at", Type: model.NanoTime(), NotNull: true, ServerOnly: true}, // nanosegundos
	},
}

var PresenceModel = model.Definition{
	Name:  "chat_presence",
	Allow: model.Create | model.Update,
	Rules: model.Rules{Create: model.AddMyBusinessRule, Update: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text(), NotNull: true, ServerOnly: true},
		{Name: "user_id", Type: model.Text(), DB: &model.FieldDB{PK: true}, NotNull: true},
		{Name: "last_seen", Type: model.NanoTime(), NotNull: true},
	},
}

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

// Definiciones de salida / resultado (Output / Result Definitions)

var ParticipantModel = model.Definition{
	Name:  "participant",
	Wire:  true,
	Allow: model.Read,
	Rules: model.Rules{Read: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "user_id", Type: model.Text(), NotNull: true},
		{Name: "label", Type: model.Text(), NotNull: true},
		{Name: "online", Type: model.Bool(), NotNull: true},
	},
}

var RoomSummaryModel = model.Definition{
	Name:  "room_summary",
	Wire:  true,
	Allow: model.Read,
	Rules: model.Rules{Read: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "room_id", Type: model.Text(), NotNull: true},
		{Name: "kind", Type: model.Text(), NotNull: true},
		{Name: "title", Type: model.Text(), NotNull: true},
		{Name: "unread", Type: model.Int(), NotNull: true},
		{Name: "last_message_at", Type: model.NanoTime(), NotNull: true},
		{Name: "last_preview", Type: model.Text(), NotNull: true},
	},
}

var MessageViewModel = model.Definition{
	Name:  "message_view",
	Wire:  true,
	Allow: model.Read,
	Rules: model.Rules{Read: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), NotNull: true},
		{Name: "room_id", Type: model.Text(), OmitEmpty: true},
		{Name: "after_id", Type: model.Text(), OmitEmpty: true},
		{Name: "sender_id", Type: model.Text(), NotNull: true},
		{Name: "sender_label", Type: model.Text(), NotNull: true},
		{Name: "body", Type: model.Text(), NotNull: true},
		{Name: "created_at", Type: model.NanoTime(), NotNull: true},
		{Name: "mine", Type: model.Bool(), NotNull: true},
		{Name: "is_read", Type: model.Bool(), NotNull: true},
	},
}

var MessageSentEventModel = model.Definition{
	Name: "message_sent_event",
	Wire: true,
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "room_id", Type: model.Text(), NotNull: true},
		{Name: "message_id", Type: model.Text(), NotNull: true},
		{Name: "sender_id", Type: model.Text(), NotNull: true},
	},
}

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

var Definition = model.ModuleDefinition{
	Name: "chat_room",
	Entities: []model.Definition{
		RoomModel, MemberModel, MessageModel, PresenceModel, DirectModel,
		ParticipantModel, RoomSummaryModel, MessageViewModel,
	},
}

type (
	notFound string
	invalid  string
)

func (e notFound) Error() string         { return string(e) }
func (e notFound) Kind() model.ErrorKind { return model.ErrorNotFound }

func (e invalid) Error() string         { return string(e) }
func (e invalid) Kind() model.ErrorKind { return model.ErrorInvalid }

// Errores (texto exacto) (Errors (exact text))
const (
	ErrNotFound          notFound = "chat_room: not found"
	ErrTenantRequired    invalid  = "chat_room: tenant_id is required"
	ErrUnauthenticated   invalid  = "chat_room: authenticated user required"
	ErrNotMember         invalid  = "chat_room: not a member of this room"
	ErrUnknownUser       invalid  = "chat_room: unknown participant"
	ErrSelfDirect        invalid  = "chat_room: cannot open a direct room with yourself"
	ErrEmptyBody         invalid  = "chat_room: message body is empty"
	ErrBodyTooLong       invalid  = "chat_room: message body is too long"
	ErrNotGroup          invalid  = "chat_room: room is not a group"
	ErrNameInvalid       invalid  = "chat_room: group name must have 1 to 60 characters"
	ErrFieldNotEditable  invalid  = "chat_room: that field cannot be changed"
	ErrRoomRequired      invalid  = "chat_room: room_id is required"
	ErrDirectPairInvalid invalid  = "chat_room: a direct pair needs user_a smaller than user_b"
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
