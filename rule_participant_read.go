package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/time"
)

func (p *Participant) Read(a crud.CurrentUser, filter *Participant, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	opts, err := m.participants.ParticipantOptions(m.tenantID)
	if err != nil {
		return nil, err
	}
	presences, err := m.getPresences()
	if err != nil {
		return nil, err
	}
	now := m.nowNano()
	list := ParticipantList{}
	for _, opt := range opts {
		var lastSeen time.Nano
		for _, pres := range presences {
			if pres.UserId == opt.Key {
				lastSeen = time.Nano(pres.LastSeen)
				break
			}
		}
		list = append(list, &Participant{
			UserId: opt.Key,
			Label:  opt.Value,
			Online: isOnline(now, lastSeen),
		})
	}
	return &list, nil
}
