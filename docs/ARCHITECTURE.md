# Arquitectura del Módulo `chat_room`

## 1. Alcance y Dominio

El módulo `chat_room` proporciona mensajería interna entre usuarios dentro de un mismo establecimiento (tenant) en el ecosistema Velty.

- **Genérico por diseño**: Administra salas tipo `direct` (1:1), `group` (miembros explícitos) y `broadcast` (General por tenant).
- **Participantes desacoplados**: Los usuarios autorizados para chatear provienen de una interfaz `ParticipantReader` (`ParticipantOptions(tenantID)`).
- **Identidad basada en sesión**: La identidad del usuario emisor/lector procede exclusivamente del contexto autenticado (`ctx.UserID()`).
- **Avisos push sin contenido**: Al enviar un mensaje se notifica a los destinatarios mediante un evento en el canal `InboxTopic(userID)` sin texto. El cliente consulta los datos mediante operaciones tipadas.
- **Retención de mensajes**: `PurgeExpired` elimina mensajes cuya antigüedad supera `Deps.RetentionDays`.

## 2. Entidades

```
chat_room (Room)
  ├── chat_member (Member)
  ├── chat_message (Message)
  └── chat_presence (Presence)
```

- **Room**: Representa una sala de chat (`id`, `tenant_id`, `kind`, `name`, `direct_key`, `created_at`, `updated_at`).
- **Member**: Relación miembro-sala (`tenant_id`, `room_id`, `user_id`, `last_read_at`).
- **Message**: Mensajes de texto enviados (`id`, `tenant_id`, `room_id`, `sender_id`, `sender_label`, `body`, `created_at`).
- **Presence**: Marca de tiempo del último latido por usuario (`tenant_id`, `user_id`, `last_seen`).

## 3. Operaciones del Módulo

| Operación | Recurso / Acción | Descripción |
|---|---|---|
| `list_participants` | Authenticated | Lista los participantes de la instalación |
| `heartbeat` | Authenticated | Registra la presencia del usuario y retorna participantes con su estado online/offline |
| `open_direct` | Authenticated | Abre o retorna la sala directa (1:1) entre el usuario actual y otro participante |
| `list_rooms` | Authenticated | Retorna el resumen de salas del usuario con número de mensajes no leídos y vista previa |
| `list_messages` | Authenticated | Lista los mensajes de una sala |
| `send_message` | Authenticated | Envía un mensaje a una sala y emite un aviso push a los miembros |
| `mark_read` | Authenticated | Actualiza la marca de lectura (`last_read_at`) del usuario en una sala |
| `list_groups` | `chat_group` / Read | Lista los grupos existentes en el tenant |
| `save_group` | `chat_group` / Create\|Update | Crea o actualiza un grupo |
| `set_group_members` | `chat_group` / Update | Actualiza los miembros de un grupo |
| `list_group_members` | `chat_group` / Read | Lista los miembros asignados a un grupo |
| `list_group_candidates` | `chat_group` / Read | Lista todos los usuarios candidatos para la asignación de miembros a un grupo |
| `delete_group` | `chat_group` / Delete | Elimina un grupo y sus mensajes/miembros |

## 4. Flujo Push y Presencia (Heartbeat)

### Flujo Push
1. El emisor llama a `send_message`.
2. El módulo guarda el mensaje, actualiza la lectura del emisor y emite un evento `InboxTopic(userID)` para cada destinatario en `Deps.Publisher`.
3. El navegador receptor escucha en su canal `InboxTopic(userID)` y ejecuta `m.view.Refresh()`, consultando las operaciones `list_rooms` y `list_messages`.

### Flujo Heartbeat
1. Cada 30 segundos (`HeartbeatIntervalSeconds`), el cliente invoca `heartbeat` (`OpHeartbeat`).
2. El servidor actualiza `last_seen` en `chat_presence` para el usuario y evalúa si otros participantes estuvieron activos en los últimos 90 segundos (`PresenceWindowSeconds`).

## 5. Ejemplo de Integración en Composición Root

### En el Servidor:
```go
db := orm.New(storageConn)
mod, err := chatroom.New(db, chatroom.Deps{
    IDs:           idGen,
    Participants:  participantsService,
    TenantID:      tenantID,
    RetentionDays: 90,
    Publisher:     broker,
})

// Migración de esquema
err = migrate.Migrate(db.RawConn().(ddl.Execer), db.RawConn().(ddl.Compiler))

// Registro de operaciones
mod.MountOperations(routerRegistry)
```

### En el Cliente (UI):
```go
caller := loopback.ActingAs(tenantID, currentUserID, mod)

// Módulo de la pantalla de chat con notificaciones push
chatModule, err := ui.Browser(caller, ids, tenantID, ui.WithInbox(subscriber, currentUserID))

// Módulo de administración de grupos
groupsModule, err := ui.GroupsBrowser(caller, ids, tenantID)
```
