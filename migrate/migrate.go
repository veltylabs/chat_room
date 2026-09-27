package migrate

import (
	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/ddl"
	"webtyp.com/storage"
)

// Sync migrates the database schema for chat_room models in order:
// Room, Member, Message, Presence.
func Sync(conn storage.Conn) error {
	compiler, ok := conn.(ddl.Compiler)
	if !ok {
		return nil
	}
	db := ddl.New(conn, compiler)
	return db.Sync(
		&chatroom.Room{},
		&chatroom.Member{},
		&chatroom.Message{},
		&chatroom.Presence{},
	)
}
