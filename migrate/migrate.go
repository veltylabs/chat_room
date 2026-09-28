package migrate

import (
	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/ddl"
)

// Migrate reconciles the schema chat_room owns: Room, Member, Message, Presence (in that order: FKs).
// Deliberately NOT called by New. Sync, not CreateTable: additive, safe to run repeatedly.
func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error {
	return ddl.New(conn, ddlCompiler).Sync(&chatroom.Room{}, &chatroom.Member{}, &chatroom.Message{}, &chatroom.Presence{})
}
