package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

func (next *Member) Update(a crud.CurrentUser, tx crud.Tx, old *Member, fields []string, m *Module) (crud.RuleAnswer, error) {
	if old.UserId != a.UserID {
		return crud.RuleAnswer{}, &model.FieldError{Field: Member_.UserId, Err: ErrNotMember}
	}
	if err := onlyFields(fields, Member_.LastReadAt); err != nil {
		return crud.RuleAnswer{}, err
	}
	next.LastReadAt = m.nowNano()
	return crud.ApplyAndPublish(), nil
}
