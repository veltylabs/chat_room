package ui

import (
	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/components/countbadge"
	"webtyp.com/dom"
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/layout/chatview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
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

func Browser(caller router.Caller, ids model.IDGenerator, tenantID string, opts ...Option) (platformd.UIModule, error) {
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

	v.OnError = func(err error) {
		if m.notifier != nil {
			m.notifier.Notify(fmt.Msg.Error, err.Error(), platformd.Auto())
		}
	}

	m.screen = &screen{
		m: m,
	}

	if o.sub != nil && o.userID != "" {
		o.sub.Subscribe(chatroom.InboxTopic(o.userID), func(events.Event) {
			m.onPush()
		})
	}

	return m, nil
}

type chatModule struct {
	id, label  string
	view       *chatview.ChatView
	screen     *screen
	badge      *countbadge.CountBadge
	notifier   platformd.Notifier
	lastUnread string
}

func (m *chatModule) ModelName() string                { return m.id }
func (m *chatModule) Label() string                    { return m.label }
func (m *chatModule) Icon() svg.Icon                   { return Icon(m.id) }
func (m *chatModule) View() dom.Component              { return m.screen }
func (m *chatModule) Badge() *countbadge.CountBadge    { return m.badge }
func (m *chatModule) UseNotifier(n platformd.Notifier) { m.notifier = n }

var _ platformd.UIModule = (*chatModule)(nil)
var _ platformd.Badged = (*chatModule)(nil)
var _ platformd.UsesNotifier = (*chatModule)(nil)

func (m *chatModule) onPush() {
	m.view.Refresh()
	m.checkUnread()
}

func (m *chatModule) checkUnread() {
	if m.badge == nil || m.badge.Count == nil {
		return
	}
	now := m.badge.Count.Get()
	if atoi(now) > atoi(m.lastUnread) && m.notifier != nil {
		m.notifier.Notify(fmt.Msg.Info, "Nuevo mensaje en el chat", platformd.Auto())
	}
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
	return dom.NewElement("div").Attr("data-chat-screen", "").Child(s.m.view)
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
