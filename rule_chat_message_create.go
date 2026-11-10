package chatroom

import "webtyp.com/crud"

func (p *Message) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if _, err := m.checkSendMessage(tx, a.UserID, p); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
