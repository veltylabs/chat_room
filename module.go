package chatroom

import (
	"webtyp.com/time"
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
)

// ParticipantReader: los usuarios que pueden chatear. Key = user_id, Value = nombre visible.
// Ejemplo: type participants struct{}; func (p participants) ParticipantOptions(tenantID string) ([]fmt.KeyValue, error) { return nil, nil }
type ParticipantReader interface {
	ParticipantOptions(tenantID string) ([]fmt.KeyValue, error)
}

type Deps struct {
	IDs           model.IDGenerator // requerido
	Participants  ParticipantReader // requerido
	TenantID      string            // requerido — tenant por defecto
	RetentionDays int               // requerido, > 0 — decisión de cada instalación, sin valor por defecto oculto
	Publisher     events.Publisher  // opcional — nil: no hay avisos push (la vista igual funciona al recargar)
	Clock         func() time.Nano      // opcional — nil = time.Now (nanosegundos); inyectable en tests
}

type Module struct {
	db            *orm.DB
	ids           model.IDGenerator
	participants  ParticipantReader
	tenantID      string
	retentionDays int
	publisher     events.Publisher
	clock         func() time.Nano
}

func New(db *orm.DB, deps Deps) (*Module, error) {
	const depPrefix = "chat_room: Deps."
	if deps.IDs == nil {
		return nil, fmt.Err(depPrefix+"IDs", "is required")
	}
	if deps.Participants == nil {
		return nil, fmt.Err(depPrefix+"Participants", "is required")
	}
	if deps.TenantID == "" {
		return nil, fmt.Err(depPrefix+"TenantID", "is required")
	}
	if deps.RetentionDays <= 0 {
		return nil, fmt.Err(depPrefix+"RetentionDays", "must be greater than zero")
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
