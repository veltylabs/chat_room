# chat_room
<img src="docs/img/badges.svg">

Generic chat rooms module for the Velty ecosystem. Provides 1:1 direct messaging, group chat administration, and broadcast rooms with real-time push event notifications and presence heartbeat tracking.

## Quick Start

```go
import (
    chatroom "github.com/veltylabs/chat_room"
    "github.com/veltylabs/chat_room/migrate"
    "github.com/veltylabs/chat_room/ui"
)

// Initialise domain module
mod, err := chatroom.New(db, chatroom.Deps{
    IDs:           idGen,
    Participants:  participantsReader,
    TenantID:      "tenant-id",
    RetentionDays: 90,
    Publisher:     eventBroker,
})

// Run database migrations
err = migrate.Migrate(dbConn, ddlCompiler)

// Mount UI modules
chatUI, err := ui.Browser(caller, idGen, "tenant-id", ui.WithInbox(eventSubscriber, currentUserID))
groupsUI, err := ui.GroupsBrowser(caller, idGen, "tenant-id")
```

## Operations Table

| Operation | Access | Description |
|---|---|---|
| `list_participants` | Authenticated | List tenant participants |
| `heartbeat` | Authenticated | Record presence & get participant status |
| `open_direct` | Authenticated | Open or fetch 1:1 direct room |
| `list_rooms` | Authenticated | List user room summaries and unread counts |
| `list_messages` | Authenticated | List room messages |
| `send_message` | Authenticated | Send message & notify recipients |
| `mark_read` | Authenticated | Mark room as read |
| `list_groups` | `chat_group` / Read | List chat groups |
| `save_group` | `chat_group` / Create\|Update | Create or rename group |
| `set_group_members` | `chat_group` / Update | Assign group members |
| `list_group_members` | `chat_group` / Read | List group members |
| `list_group_candidates` | `chat_group` / Read | List candidate users for group assignment |
| `delete_group` | `chat_group` / Delete | Delete group and messages |

## Demo

To run the interactive browser demo:

```bash
webtyp
```
