# chat_room
<img src="docs/img/badges.svg">

Módulo de salas de chat genérico para el ecosistema Velty. Proporciona mensajería directa 1:1, administración de chat de grupos y salas de transmisión (broadcast) con notificaciones de eventos push en tiempo real y seguimiento de presencia (heartbeat).

## Inicio Rápido

```go
import (
    chatroom "github.com/veltylabs/chat_room"
    "github.com/veltylabs/chat_room/migrate"
    "github.com/veltylabs/chat_room/ui"
)

// Inicializa el módulo de dominio
mod, err := chatroom.New(db, chatroom.Deps{
    IDs:           idGen,
    Participants:  participantsReader,
    TenantID:      "tenant-id",
    RetentionDays: 90,
    Publisher:     eventBroker,
})

// Ejecuta las migraciones de base de datos
err = migrate.Migrate(dbConn, ddlCompiler)

// Monta los módulos de interfaz de usuario
screens := ui.Screens(caller, idGen, "tenant-id", ui.WithInbox(eventSubscriber, currentUserID))
```

## Tabla de Operaciones

| Operación | Acceso | Descripción |
|---|---|---|
| `list_participants` | Authenticated | Lista los participantes del tenant |
| `heartbeat` | Authenticated | Registra la presencia y obtiene el estado de los participantes |
| `open_direct` | Authenticated | Abre o recupera una sala directa 1:1 |
| `list_rooms` | Authenticated | Lista el resumen de salas del usuario y recuentos de no leídos |
| `list_messages` | Authenticated | Lista los mensajes de una sala |
| `send_message` | Authenticated | Envía un mensaje y notifica a los destinatarios |
| `mark_read` | Authenticated | Marca una sala como leída |
| `list_groups` | `chat_group` / Read | Lista los grupos de chat |
| `save_group` | `chat_group` / Create\|Update | Crea o renombra un grupo |
| `set_group_members` | `chat_group` / Update | Asigna miembros a un grupo |
| `list_group_members` | `chat_group` / Read | Lista los miembros de un grupo |
| `list_group_candidates` | `chat_group` / Read | Lista los usuarios candidatos para asignar a un grupo |
| `delete_group` | `chat_group` / Delete | Elimina un grupo y sus mensajes |

## Demo

Para ejecutar la demo interactiva en el navegador:

```bash
webtyp
```
