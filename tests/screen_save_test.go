package tests

import (
	"github.com/veltylabs/chat_room"
	"testing"
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/view"
)

func TestScreen_SaveAndDeleteGroup(t *testing.T) {
	env, _ := setupTestEnv("t1", 30, nil)

	recs := view.NewRecords()
	caller := recs.Caller(crud.LoopbackActingAs(nil, "user-1", env.module))
	pres := chatroom.NewGroupView(caller)

	var saveErr error
	pres.(view.Saver).Save([]model.Model{&chatroom.Room{
		Id:   "g1",
		Kind: chatroom.KindGroup,
		Name: "General",
	}}, func(err error) { saveErr = err })
	if saveErr != nil {
		t.Fatalf("save failed: %v", saveErr)
	}

	items := pres.Items()
	if len(items) == 0 {
		t.Fatalf("expected saved room in presenter items")
	}
	id := items[0].ID

	var delErr error
	pres.(view.Deleter).Delete([]string{id}, func(err error) { delErr = err })
	if delErr != nil {
		t.Fatalf("delete failed: %v", delErr)
	}
}
