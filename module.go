package chatroom

import (
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
)

// ParticipantReader: los usuarios que pueden chatear. Key = user_id, Value = nombre visible.
type ParticipantReader interface {
	ParticipantOptions(tenantID string) ([]fmt.KeyValue, error)
}

type Deps struct {
	IDs           model.IDGenerator // requerido
	Participants  ParticipantReader // requerido
	TenantID      string            // requerido — tenant por defecto
	RetentionDays int               // requerido, > 0 — decisión de cada instalación, sin valor por defecto oculto
	Publisher     events.Publisher  // opcional — nil: no hay avisos push (la vista igual funciona al recargar)
	Clock         func() int64      // opcional — nil = time.Now (nanosegundos); inyectable en tests
}

type Module struct {
	db            *orm.DB
	ids           model.IDGenerator
	participants  ParticipantReader
	tenantID      string
	retentionDays int
	publisher     events.Publisher
	clock         func() int64
}

func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err("chat_room: Deps.IDs is required")
	}
	if deps.Participants == nil {
		return nil, fmt.Err("chat_room: Deps.Participants is required")
	}
	if deps.TenantID == "" {
		return nil, fmt.Err("chat_room: Deps.TenantID is required")
	}
	if deps.RetentionDays <= 0 {
		return nil, fmt.Err("chat_room: Deps.RetentionDays must be greater than zero")
	}

	return &Module{
		db:            db,
		ids:           deps.IDs,
		participants:  deps.Participants,
		tenantID:      deps.TenantID,
		retentionDays: deps.RetentionDays,
		publisher:     deps.Publisher,
		clock:         deps.Clock,
	}, nil
}

func (m *Module) ModelName() string {
	return ModelName
}
