package ui

import (
	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/components/bubblethread"
	"webtyp.com/components/inboxlist"
	"webtyp.com/components/presencelist"
	"webtyp.com/layout/chatview"
	"webtyp.com/router"
	"webtyp.com/time"
)

type source struct {
	caller   router.Caller
	tenantID string
}

var _ chatview.Source = (*source)(nil)

func (s *source) Rooms(done func(rows []inboxlist.Row, err error)) {
	args := chatroom.ListRoomsArgs{TenantId: s.tenantID}
	var res chatroom.RoomSummaryList
	s.caller.Call(chatroom.ModelName+"."+chatroom.OpListRooms, &args, &res, func(err error) {
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
				Time:    shortTime(sum.LastMessageAt),
				Unread:  int(sum.Unread),
			}
		}
		done(rows, nil)
	})
}

func (s *source) Messages(roomID string, done func(bubbles []bubblethread.Bubble, err error)) {
	args := chatroom.ListMessagesArgs{TenantId: s.tenantID, RoomId: roomID}
	var res chatroom.MessageViewList
	s.caller.Call(chatroom.ModelName+"."+chatroom.OpListMessages, &args, &res, func(err error) {
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
				Time:   clock(msg.CreatedAt),
				Mine:   msg.Mine,
				Read:   msg.Read,
			}
		}
		done(bubbles, nil)
	})
}

func (s *source) Send(roomID, body string, done func(sent bubblethread.Bubble, err error)) {
	args := chatroom.SendMessageArgs{TenantId: s.tenantID, RoomId: roomID, Body: body}
	var sent chatroom.MessageView
	s.caller.Call(chatroom.ModelName+"."+chatroom.OpSendMessage, &args, &sent, func(err error) {
		if err != nil {
			done(bubblethread.Bubble{}, err)
			return
		}
		b := bubblethread.Bubble{
			ID:     sent.Id,
			Author: sent.SenderLabel,
			Body:   sent.Body,
			Time:   clock(sent.CreatedAt),
			Mine:   sent.Mine,
			Read:   sent.Read,
		}
		done(b, nil)
	})
}

func (s *source) MarkRead(roomID string, done func(err error)) {
	args := chatroom.MarkReadArgs{TenantId: s.tenantID, RoomId: roomID}
	s.caller.Call(chatroom.ModelName+"."+chatroom.OpMarkRead, &args, nil, done)
}

func (s *source) People(done func(people []presencelist.Person, err error)) {
	args := chatroom.HeartbeatArgs{TenantId: s.tenantID}
	var res chatroom.ParticipantList
	s.caller.Call(chatroom.ModelName+"."+chatroom.OpHeartbeat, &args, &res, func(err error) {
		if err != nil {
			done(nil, err)
			return
		}
		people := make([]presencelist.Person, res.Len())
		for i := 0; i < res.Len(); i++ {
			p := res.At(i).(*chatroom.Participant)
			people[i] = presencelist.Person{
				ID:     p.UserId,
				Label:  p.Label,
				Online: p.Online,
			}
		}
		done(people, nil)
	})
}

func (s *source) OpenDirect(personID string, done func(roomID string, err error)) {
	args := chatroom.OpenDirectArgs{TenantId: s.tenantID, OtherUserId: personID}
	var res chatroom.RoomSummary
	s.caller.Call(chatroom.ModelName+"."+chatroom.OpOpenDirect, &args, &res, func(err error) {
		if err != nil {
			done("", err)
			return
		}
		done(res.RoomId, nil)
	})
}

func clock(ns int64) string {
	if ns == 0 {
		return ""
	}
	t := time.FormatTime(ns)
	if len(t) >= 5 {
		return t[:5]
	}
	return t
}

func shortTime(ns int64) string {
	if ns == 0 {
		return ""
	}
	if time.IsToday(ns) {
		return clock(ns)
	}
	return time.FormatDate(ns)
}
