package tests

import (
	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/events/mock"
	"webtyp.com/fmt"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
)

type mockIDGen struct {
	counter int
}

func (g *mockIDGen) NewID() string {
	g.counter++
	return fmt.Sprintf("id-%d", g.counter)
}

type mockParticipants struct {
	list []fmt.KeyValue
}

func (p *mockParticipants) ParticipantOptions(tenantID string) ([]fmt.KeyValue, error) {
	return p.list, nil
}

type testEnv struct {
	db        *orm.DB
	idGen     *mockIDGen
	parts     *mockParticipants
	broker    *mock.Broker
	clockTime int64
	module    *chatroom.Module
}

func setupTestEnv(tenantID string, retentionDays int, participantList []fmt.KeyValue) (*testEnv, error) {
	memConn := mem.New()
	db := orm.New(memConn)
	idGen := &mockIDGen{}
	parts := &mockParticipants{list: participantList}
	broker := &mock.Broker{}

	env := &testEnv{
		db:        db,
		idGen:     idGen,
		parts:     parts,
		broker:    broker,
		clockTime: 1_000_000_000_000,
	}

	deps := chatroom.Deps{
		IDs:           idGen,
		Participants:  parts,
		TenantID:      tenantID,
		RetentionDays: retentionDays,
		Publisher:     broker,
		Clock:         func() int64 { return env.clockTime },
	}

	m, err := chatroom.New(db, deps)
	if err != nil {
		return nil, err
	}
	env.module = m

	return env, nil
}
