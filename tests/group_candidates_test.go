package tests

import (
	"testing"

	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/fmt"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
)

type tenantAwareParticipants struct {
	a []fmt.KeyValue
	b []fmt.KeyValue
}

func (p *tenantAwareParticipants) ParticipantOptions(tenantID string) ([]fmt.KeyValue, error) {
	if tenantID == "tenant-b" {
		return p.b, nil
	}
	return p.a, nil
}

func TestListGroupCandidates(t *testing.T) {
	partsA := []fmt.KeyValue{
		{Key: "u1", Value: "Alice"},
		{Key: "u2", Value: "Bob"},
		{Key: "admin", Value: "Admin User"},
	}
	partsB := []fmt.KeyValue{
		{Key: "tb_u1", Value: "Tenant B User"},
	}

	parts := &tenantAwareParticipants{a: partsA, b: partsB}
	memConn := mem.New()
	db := orm.New(memConn)
	idGen := &mockIDGen{}

	m, err := chatroom.New(db, chatroom.Deps{
		IDs:           idGen,
		Participants:  parts,
		TenantID:      "tenant-a",
		RetentionDays: 30,
	})
	if err != nil {
		t.Fatalf("chatroom.New failed: %v", err)
	}

	// 1. ListGroupCandidates returns everyone including admin
	candidates, err := m.ListGroupCandidates("tenant-a")
	if err != nil {
		t.Fatalf("ListGroupCandidates failed: %v", err)
	}
	if len(candidates) != 3 {
		t.Fatalf("expected 3 candidates, got %d", len(candidates))
	}
	foundAdmin := false
	for _, c := range candidates {
		if c.UserId == "admin" {
			foundAdmin = true
			break
		}
	}
	if !foundAdmin {
		t.Fatalf("expected admin user in group candidates list")
	}

	// 2. Multi-tenant isolation: tenant-b candidates are distinct
	candidatesB, err := m.ListGroupCandidates("tenant-b")
	if err != nil {
		t.Fatalf("ListGroupCandidates tenant-b failed: %v", err)
	}
	if len(candidatesB) != 1 || candidatesB[0].UserId != "tb_u1" {
		t.Fatalf("unexpected candidates for tenant-b: %v", candidatesB)
	}
}
