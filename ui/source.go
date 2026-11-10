package ui

import (
	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/components/bubblethread"
	"webtyp.com/components/inboxlist"
	"webtyp.com/components/presencelist"
	"webtyp.com/crud"
	"webtyp.com/layout/chatview"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/time"
)

type source struct {
	caller   router.Caller
	zone     time.Zone
	tenantID string
	userID   string
}

var _ chatview.Source = (*source)(nil)

func (s *source) Rooms(done func(rows []inboxlist.Row, err error)) {
	var res chatroom.RoomSummaryList
	s.caller.Call(
		router.Operation{Module: chatroom.ModelName, Entity: "room_summary", Action: model.Read},
		&crud.ReadArgs{},
		&res,
		func(err error) {
			if err != nil {
				done(nil, err)
				return
			}
			rows := make([]inboxlist.Row, res.Len())
			for i := 0; i < res.Len(); i++ {
				sum := res.At(i).(*chatroom.RoomSummary)
				rows[i] = inboxlist.Row{
					ID:      sum.RoomId,
					Title:   sum.Title,
					Preview: sum.LastPreview,
					Time:    s.shortTime(sum.LastMessageAt),
					Unread:  int(sum.Unread),
				}
			}
			done(rows, nil)
		},
	)
}

func (s *source) Messages(roomID string, done func(bubbles []bubblethread.Bubble, err error)) {
	var res chatroom.MessageViewList
	s.caller.Call(
		router.Operation{Module: chatroom.ModelName, Entity: "message_view", Action: model.Read},
		&crud.ReadArgs{Filter: &chatroom.MessageView{RoomId: roomID}},
		&res,
		func(err error) {
			if err != nil {
				done(nil, err)
				return
			}
			bubbles := make([]bubblethread.Bubble, res.Len())
			for i := 0; i < res.Len(); i++ {
				msg := res.At(i).(*chatroom.MessageView)
				bubbles[i] = bubblethread.Bubble{
					ID:     msg.Id,
					Author: msg.SenderLabel,
					Body:   msg.Body,
					Time:   s.clock(msg.CreatedAt),
					Mine:   msg.Mine,
					Read:   msg.IsRead,
				}
			}
			done(bubbles, nil)
		},
	)
}

func (s *source) Send(roomID, body string, done func(sent bubblethread.Bubble, err error)) {
	var sent chatroom.Message
	s.caller.Call(
		router.Operation{Module: chatroom.ModelName, Entity: "chat_message", Action: model.Create},
		&crud.CreateArgs{Records: []model.Model{&chatroom.Message{RoomId: roomID, Body: body}}},
		&sent,
		func(err error) {
			if err != nil {
				done(bubblethread.Bubble{}, err)
				return
			}
			b := bubblethread.Bubble{
				ID:     sent.Id,
				Author: sent.SenderLabel,
				Body:   sent.Body,
				Time:   s.clock(sent.CreatedAt),
				Mine:   true,
				Read:   false,
			}
			done(b, nil)
		},
	)
}

func (s *source) MarkRead(roomID string, done func(err error)) {
	if s.userID == "" {
		done(nil)
		return
	}
	s.caller.Call(
		router.Operation{Module: chatroom.ModelName, Entity: "chat_member", Action: model.Update},
		&crud.UpdateArgs{
			IDs:    []string{model.JoinKey(roomID, s.userID)},
			Fields: []string{chatroom.Member_.LastReadAt},
			Record: &chatroom.Member{LastReadAt: 1},
		},
		nil,
		done,
	)
}

func (s *source) People(done func(people []presencelist.Person, err error)) {
	fetch := func() {
		var res chatroom.ParticipantList
		s.caller.Call(
			router.Operation{Module: chatroom.ModelName, Entity: "participant", Action: model.Read},
			&crud.ReadArgs{},
			&res,
			func(err error) {
				if err != nil {
					done(nil, err)
					return
				}
				people := make([]presencelist.Person, 0, res.Len())
				for i := 0; i < res.Len(); i++ {
					p := res.At(i).(*chatroom.Participant)
					if s.userID != "" && p.UserId == s.userID {
						continue
					}
					people = append(people, presencelist.Person{
						ID:     p.UserId,
						Label:  p.Label,
						Online: p.Online,
					})
				}
				done(people, nil)
			},
		)
	}

	if s.userID == "" {
		fetch()
		return
	}

	s.caller.Call(
		router.Operation{Module: chatroom.ModelName, Entity: "chat_presence", Action: model.Update},
		&crud.UpdateArgs{
			IDs:    []string{s.userID},
			Fields: []string{chatroom.Presence_.LastSeen},
			Record: &chatroom.Presence{LastSeen: 1},
		},
		nil,
		func(err error) {
			if err != nil && model.KindOf(err) == model.ErrorNotFound {
				s.caller.Call(
					router.Operation{Module: chatroom.ModelName, Entity: "chat_presence", Action: model.Create},
					&crud.CreateArgs{Records: []model.Model{&chatroom.Presence{UserId: s.userID, LastSeen: 1}}},
					nil,
					func(_ error) {
						fetch()
					},
				)
				return
			}
			fetch()
		},
	)
}

func (s *source) OpenDirect(personID string, done func(roomID string, err error)) {
	if s.userID == "" {
		done("", chatroom.ErrUnauthenticated)
		return
	}
	userA, userB := s.userID, personID
	if userA > userB {
		userA, userB = userB, userA
	}
	var direct chatroom.Direct
	s.caller.Call(
		router.Operation{Module: chatroom.ModelName, Entity: "chat_direct", Action: model.Create},
		&crud.CreateArgs{Records: []model.Model{&chatroom.Direct{UserA: userA, UserB: userB}}},
		&direct,
		func(err error) {
			if err == nil {
				done(direct.RoomId, nil)
				return
			}
			if model.KindOf(err) == model.ErrorConflict {
				var directList chatroom.DirectList
				s.caller.Call(
					router.Operation{Module: chatroom.ModelName, Entity: "chat_direct", Action: model.Read},
					&crud.ReadArgs{ID: model.JoinKey(userA, userB)},
					&directList,
					func(readErr error) {
						if readErr != nil || directList.Len() == 0 {
							done("", err)
							return
						}
						done(directList.At(0).(*chatroom.Direct).RoomId, nil)
					},
				)
				return
			}
			done("", err)
		},
	)
}

func (s *source) clock(ns time.Nano) string {
	if ns == 0 {
		return ""
	}
	return s.zone.Clock(ns.Unix())
}

func (s *source) shortTime(ns time.Nano) string {
	if ns == 0 {
		return ""
	}
	if s.zone.IsToday(ns.Unix()) {
		return s.clock(ns)
	}
	return s.zone.Date(ns.Unix())
}
