//go:build browser

package ui

import (
	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/components/countbadge"
	"webtyp.com/dom"
	"webtyp.com/events"
	"webtyp.com/layout/chatview"
	"webtyp.com/router"
	"webtyp.com/svg"
	"webtyp.com/time"
)

type options struct {
	label  string
	sub    events.Subscriber
	userID string
}

type Option func(*options)

func WithLabel(label string) Option {
	return func(o *options) {
		o.label = label
	}
}

func WithInbox(sub events.Subscriber, userID string) Option {
	return func(o *options) {
		o.sub = sub
		o.userID = userID
	}
}

func NewChatComponent(caller router.Caller, tenantID string, opts ...Option) (dom.Component, error) {
	o := &options{
		label: DefaultLabel,
	}
	for _, opt := range opts {
		opt(o)
	}

	v, err := chatview.New(chatview.Config{
		Source:        &source{caller: caller, tenantID: tenantID},
		MaxBodyLength: chatroom.MaxBodyLength,
	})
	if err != nil {
		return nil, err
	}

	count, visible := v.Unread()
	badge := &countbadge.CountBadge{Count: count, Visible: visible}

	m := &chatModule{
		id:    ID,
		label: o.label,
		view:  v,
		badge: badge,
	}

	m.screen = &screen{
		m: m,
	}

	if o.sub != nil && o.userID != "" {
		o.sub.Subscribe(chatroom.InboxTopic(o.userID), func(events.Event) {
			m.onPush()
		})
	}

	return m.screen, nil
}

type chatModule struct {
	id, label  string
	view       *chatview.ChatView
	screen     *screen
	badge      *countbadge.CountBadge
	lastUnread string
}

func (m *chatModule) ModelName() string                { return m.id }
func (m *chatModule) Label() string                    { return m.label }
func (m *chatModule) Icon() svg.Icon                   { return Icon(m.id) }
func (m *chatModule) View() dom.Component              { return m.screen }
func (m *chatModule) Activate()                        { m.onPush() }
func (m *chatModule) Badge() *countbadge.CountBadge    { return m.badge }

func (m *chatModule) onPush() {
	m.view.Refresh()
	m.checkUnread()
}

func (m *chatModule) checkUnread() {
	if m.badge == nil || m.badge.Count == nil {
		return
	}
	now := m.badge.Count.Get()
	m.lastUnread = now
}

func atoi(s string) int {
	var n int
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			n = n*10 + int(s[i]-'0')
		}
	}
	return n
}

type screen struct {
	dom.Element
	m     *chatModule
	timer time.Timer
}

func (s *screen) Render() *dom.Element {
	return dom.NewElement("div").
		Attr("data-chat-screen", "").
		Attr("style", "height:100%;min-height:0;display:flex;flex-direction:column;overflow:hidden;").
		Child(s.m.view)
}

func (s *screen) Badge() *countbadge.CountBadge {
	return s.m.Badge()
}

func (s *screen) Init(ctx dom.Ctx) {
	s.tick()
	if ctx != nil {
		ctx.OnCleanup(func() {
			if s.timer != nil {
				s.timer.Stop()
			}
		})
	}
}

func (s *screen) tick() {
	s.m.view.RefreshPeople()
	s.m.checkUnread()
	s.timer = time.AfterFunc(chatroom.HeartbeatIntervalSeconds*1000, s.tick)
}
