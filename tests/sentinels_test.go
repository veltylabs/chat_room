package tests

import (
	"testing"

	chatroom "github.com/veltylabs/chat_room"
)

func TestSentinelErrors(t *testing.T) {
	sentinels := []struct {
		err  error
		text string
	}{
		{chatroom.ErrNotFound, "chat_room: not found"},
		{chatroom.ErrTenantRequired, "chat_room: tenant_id is required"},
		{chatroom.ErrUnauthenticated, "chat_room: authenticated user required"},
		{chatroom.ErrNotMember, "chat_room: not a member of this room"},
		{chatroom.ErrUnknownUser, "chat_room: unknown participant"},
		{chatroom.ErrSelfDirect, "chat_room: cannot open a direct room with yourself"},
		{chatroom.ErrEmptyBody, "chat_room: message body is empty"},
		{chatroom.ErrBodyTooLong, "chat_room: message body is too long"},
		{chatroom.ErrNotGroup, "chat_room: room is not a group"},
	}

	for _, tc := range sentinels {
		if tc.err.Error() != tc.text {
			t.Errorf("expected %q, got %q", tc.text, tc.err.Error())
		}
	}
}
