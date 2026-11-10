package chatroom

import "webtyp.com/crud"

// StoreRoom is the store of the rooms.
func (m *Module) StoreRoom() crud.Store { return m.store }

// StoreMember is the store of the room members.
func (m *Module) StoreMember() crud.Store { return m.store }

// StoreMessage is the store of the messages.
func (m *Module) StoreMessage() crud.Store { return m.store }

// StorePresence is the store of the presence rows.
func (m *Module) StorePresence() crud.Store { return m.store }

// StoreDirect is the store of the direct pairs.
func (m *Module) StoreDirect() crud.Store { return m.store }
