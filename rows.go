package chatroom

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

// firstRow answers the first row that matches conds, and whether there is one.
// It reads through tx (a rule) or through the store (a reader): both are a crud.Tx.
//
// Example: row, found, err := firstRow(tx, func() model.Model { return &Member{} }, storage.Eq(Member_.RoomId, id))
func firstRow(tx crud.Tx, newRow func() model.Model, conds ...storage.Condition) (model.Model, bool, error) {
	var first model.Model
	err := tx.ReadAll(newRow, crud.Query{Conditions: conds, Limit: 1}, func(r model.Model) {
		if first == nil {
			first = r
		}
	})
	return first, first != nil, err
}

// allRows answers every row that matches conds, in the order asked (at most crud.MaxPage rows).
//
// Example: rows, err := allRows(tx, func() model.Model { return &Member{} }, nil, storage.Eq(Member_.RoomId, id))
func allRows(tx crud.Tx, newRow func() model.Model, order []storage.Order, conds ...storage.Condition) ([]model.Model, error) {
	var rows []model.Model
	err := tx.ReadAll(newRow, crud.Query{Conditions: conds, Order: order, Limit: crud.MaxPage}, func(r model.Model) {
		rows = append(rows, r)
	})
	return rows, err
}
