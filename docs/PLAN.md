---
PLAN: "feat: chat_room — salas, mensajes persistidos, leídos, presencia y aviso por buzón"
TAG: v0.1.0
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 17684538278888635723
PR: https://github.com/veltylabs/chat_room/pull/1
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

# Plan — `chat_room` v0.1.0 (dominio + ops)

Módulo nuevo. Este plan entrega **el dominio, las ops, el seed y los tests**. La pantalla (`ui/`) y
el demo en el navegador (`web/`) van en un **segundo plan**, porque dependen del layout
`webtyp.com/layout/chatview`, que todavía no existe.

Lee **`AGENTS.md`** en la raíz del repo antes de escribir código. Las reglas críticas se repiten en §1.

---

## 0. Contexto

Mensajería interna **entre usuarios del personal** de un establecimiento (primer consumidor: un
consultorio médico en red local). Reemplaza un chat antiguo (pa100t, Go + JavaScript) que tenía
tres tipos de destino (privado, a todos, por área), una lista de conectados y un aviso parpadeante
de mensajes nuevos. Ese chat **guardaba los mensajes solo en memoria**: se perdían al recargar.

Decisiones ya tomadas (no se reabren en este plan):

| Tema | Decisión | Por qué |
|---|---|---|
| Quién chatea | Solo usuarios del personal (ids de usuario de `auth`). **Pacientes no.** | Los pacientes no tienen cuenta; el login es RUT + equipo asignado en red local. Abrir el chat a pacientes es otro proyecto (identidad, exposición a internet, consentimiento). |
| Tipos de sala | `direct` (1:1), `group` (miembros explícitos, la crea un administrador: así se modelan las "áreas") y `broadcast` (una por tenant, "General", todos los participantes) | Cubre los tres destinos de pa100t sin atar el módulo a "áreas" o "especialidades": un spa o un hotel crean los grupos que quieran. |
| Historial | Se guarda en la base de datos y se **purga** pasado `Deps.RetentionDays` | Recargar la página no puede borrar mensajes, y en un consultorio los mensajes mencionarán pacientes (dato sensible): no deben guardarse para siempre. |
| Quién lee | **Solo los miembros de la sala.** Ningún rol, ni `admin`, lee salas ajenas | Privacidad de las conversaciones del personal. |
| Leídos | Sí: contador de no leídos por sala y marca "leído" en los mensajes propios de una sala `direct` | Es lo que el personal necesita para saber si el aviso llegó. |
| Adjuntos / "escribiendo…" | No | Almacenamiento, tamaño y datos sensibles; "escribiendo" requiere un canal de subida que no aporta en una red local. |
| Presencia | Latido (`heartbeat`) cada 30 s; en línea = último latido hace menos de 90 s | No depende del transporte, no necesita gorutinas en el módulo, y el cierre de sesión por inactividad (30 min) ya la acota. |
| Tiempo real | Tras cada mensaje, el módulo publica un **aviso sin contenido** en el buzón de cada destinatario (`InboxTopic(userID)`); la vista, al recibirlo, pide los mensajes por la op normal | Ningún texto de mensaje viaja por el broker; el navegador recibe datos tipados por el mismo camino (`router.Caller`) que el resto de la app, sin decodificar payloads push. |

---

## 1. Reglas no negociables (repetidas de `AGENTS.md`)

- Paquete raíz `package chatroom`: importa **solo** `webtyp.com/model`, `router`, `view`,
  `events`, `orm`, `storage`, `input`, `fmt`, `time`. `migrate/` además `webtyp.com/ddl`.
- **Prohibido en cualquier archivo, tests incluidos**: drivers (`sqlite`, `sqlt`, `postgres`,
  `indexdb`, `database/sql`), transportes (`mcp`, `server`, `net/http`), `unixid`, codecs
  (`webtyp.com/json`, `jsvalue`, `encoding/json`), renderers (`layout`) y cualquier
  `github.com/veltylabs/*`.

| Prohibido (stdlib) | Usar |
|---|---|
| `errors`, `strings`, `strconv`, `fmt` | `webtyp.com/fmt` |
| `time` | `webtyp.com/time` |
| `encoding/json` | modelos `ormc` (`model.Encodable`/`Decodable`) |
| `database/sql` | `webtyp.com/orm` |
| `map[K]V` | slice de structs con búsqueda lineal, o `[]fmt.KeyValue` |
| `reflect` | nada |

- **Cuál build decide:** `gotest` corre nativo + navegador (TinyGo); **ambos** deben pasar.
  `GOOS=js GOARCH=wasm go build` no prueba nada (stdlib completa, no es TinyGo).
- Forma exacta de un módulo: copiar <https://github.com/veltylabs/patient_directory>.
  `MountOperations(reg router.OperationRegistry)` con `reg.Operation(...)`; esquema en `migrate/`
  con `ddl.Sync`, **nunca** en `New`; `model_orm.go` generado por `ormc` (`go install
  webtyp.com/ormc/cmd/ormc@latest`), nunca editado a mano.
- Handler: decode (400) → `Validate` (400) → servicio → encode. 400 validación · 403 no miembro ·
  404 no encontrado · 409 conflicto · 500 solo error interno. Un error de BD nunca se vuelve 404.
- Todo UPDATE/DELETE lleva `tenant_id` en la condición. Valores cerrados = constantes exportadas.
- Tests en `tests/` (sin `go.mod` propio), sobre `orm.New(mem.New())`, ops sobre
  `webtyp.com/router/mock`, eventos sobre `webtyp.com/events/mock`. Correr con `gotest`.
- **La identidad del usuario sale de `ctx.UserID()`**, nunca de los argumentos. Ninguna op acepta
  un `sender_id` o un `user_id` "propio" en sus args.

---

## 2. Design gate

**1. Prior art.**
- *Slack*: canales públicos/privados, mensajes directos, no leídos por canal, presencia.
- *Matrix*: salas con miembros explícitos, *read receipts* por usuario, historial con
  retención configurable por servidor.
- *Rocket.Chat / Mattermost* (autohospedados): DM + grupos + canal general, retención configurable
  por política.

Este módulo toma el trío DM / grupo / general de los tres, la marca de leído de Matrix (como
"último leído" por miembro) y la retención obligatoria de Rocket.Chat/Mattermost. Difiere en el
tiempo real: no hay websocket propio ni evento con el texto. Solo se avisa "hay algo nuevo" al
buzón del usuario, y los datos se piden por la op tipada, porque el ecosistema ya tiene
`events.Publisher` en el servidor y `router.Caller` en el cliente y no hace falta un tercer camino.

**2. Prueba del nombre para novatos.** `SendMessage`, `OpenDirect`, `ListRooms`, `ListMessages`,
`MarkRead`, `Heartbeat`, `SaveGroup`, `SetGroupMembers`, `PurgeExpired`, `InboxTopic`: todos se
leen como frase.

**3. Balance de complejidad.**
```
Conceptos nuevos                 +5 (sala, miembro, mensaje, presencia, buzón)
Archivos que toca la app         +1 línea en modules/server.go, +1 adaptador de participantes, +canal SSE (ver MASTER)
Formas de hacer lo mismo          0 (no existe chat en el ecosistema)
```

**4. Dónde va.** Repo propio `veltylabs/chat_room`. Los participantes se leen mediante
`ParticipantReader`, **declarada aquí**; la app la conecta (en el consultorio, desde
`staff_manager`). No importa ningún otro módulo.

**5. Qué borra.** El esqueleto de `gonew` (`chat_room.go` con su `struct` vacío y `New()`).

---

## 3. Modelo — `model.go`

```go
package chatroom

const ModelName = "chat_room"

// Recursos RBAC (solo para administrar grupos; chatear exige solo sesión).
const ResourceGroup = "chat_group"

// Tipos de sala — únicos valores de chat_room.kind.
const (
	KindDirect    = "direct"
	KindGroup     = "group"
	KindBroadcast = "broadcast"
)

const (
	BroadcastName             = "General"
	MaxBodyLength             = 2000
	PreviewLength             = 80
	HeartbeatIntervalSeconds  = 30 // lo usa la vista
	PresenceWindowSeconds     = 90
	DefaultMessagesPageLimit  = 50
)

var RoomModel = model.Definition{
	Name: "chat_room",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "kind", Type: model.Text(), NotNull: true},
		{Name: "name", Type: model.Text(), OmitEmpty: true},       // solo group y broadcast
		{Name: "direct_key", Type: model.Text(), OmitEmpty: true}, // solo direct: "<idMenor>|<idMayor>"
		{Name: "created_at", Type: model.Int(), NotNull: true},
		{Name: "updated_at", Type: model.Int(), OmitEmpty: true},
	},
}

var MemberModel = model.Definition{
	Name: "chat_member",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "room_id", Type: model.Text(), Ref: &RoomModel, DB: &model.FieldDB{PK: true, RefColumn: "id"}, NotNull: true},
		{Name: "user_id", Type: model.Text(), DB: &model.FieldDB{PK: true}, NotNull: true},
		{Name: "last_read_at", Type: model.Int()},
	},
}

var MessageModel = model.Definition{
	Name: "chat_message",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "room_id", Type: model.Text(), Ref: &RoomModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "sender_id", Type: model.Text(), NotNull: true},
		{Name: "sender_label", Type: model.Text(), NotNull: true}, // copia del nombre al enviar
		{Name: "body", Type: model.Text(), NotNull: true},
		{Name: "created_at", Type: model.Int(), NotNull: true},    // time.Now(), nanosegundos
	},
}

var PresenceModel = model.Definition{
	Name: "chat_presence",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "user_id", Type: model.Text(), DB: &model.FieldDB{PK: true}, NotNull: true},
		{Name: "last_seen", Type: model.Int(), NotNull: true},
	},
}
```

Argumentos (sin widgets salvo donde se indica; ninguno lleva identidad del que llama):

| Definition | Campos |
|---|---|
| `open_direct_args` | `tenant_id`, `other_user_id` |
| `list_rooms_args` | `tenant_id` |
| `list_messages_args` | `tenant_id`, `room_id`, `after_id` (vacío = desde el principio de la ventana), `limit` (Int; 0 = `DefaultMessagesPageLimit`) |
| `send_message_args` | `tenant_id`, `room_id`, `body` (`input.Textarea()`, `NotNull`, `Permitted{Minimum: 1, Maximum: MaxBodyLength}`) |
| `mark_read_args` | `tenant_id`, `room_id` |
| `heartbeat_args` | `tenant_id` |
| `list_groups_args` | `tenant_id` |
| `save_group_args` | `tenant_id`, `id` (vacío crea), `name` (`input.Text()`, `NotNull`, `Permitted{Minimum: 1, Maximum: 60}`) |
| `set_group_members_args` | `tenant_id`, `room_id`, `user_ids` (`model.StructSlice(&IdRefModel)`) |
| `list_group_members_args` | `tenant_id`, `room_id` |
| `delete_group_args` | `tenant_id`, `id` |

`IdRefModel` = `{Name: "id_ref", Fields: {{Name: "id", Type: model.Text(), NotNull: true}}}`.

Salidas (solo `model.*`):

| Definition | Campos |
|---|---|
| `participant` | `user_id`, `label`, `online` (Bool) |
| `room_summary` | `room_id`, `kind`, `title` (direct: el nombre del otro; group/broadcast: `name`), `unread` (Int), `last_message_at` (Int), `last_preview` (primeros `PreviewLength` caracteres) |
| `message_view` | `id`, `room_id`, `sender_id`, `sender_label`, `body`, `created_at`, `mine` (Bool), `read` (Bool: solo en `direct` y solo en mensajes propios, `true` si `last_read_at` del otro miembro ≥ `created_at`) |

Errores (texto exacto):

```go
var (
	ErrNotFound          = fmt.Err("chat_room: not found")
	ErrTenantRequired    = fmt.Err("chat_room: tenant_id is required")
	ErrUnauthenticated   = fmt.Err("chat_room: authenticated user required")
	ErrNotMember         = fmt.Err("chat_room: not a member of this room")
	ErrUnknownUser       = fmt.Err("chat_room: unknown participant")
	ErrSelfDirect        = fmt.Err("chat_room: cannot open a direct room with yourself")
	ErrEmptyBody         = fmt.Err("chat_room: message body is empty")
	ErrBodyTooLong       = fmt.Err("chat_room: message body is too long")
	ErrNotGroup          = fmt.Err("chat_room: room is not a group")
)
```

Estados: `ErrUnauthenticated` → 401 · `ErrNotMember` → 403 · `ErrNotFound` → 404 · `ErrUnknownUser`,
`ErrSelfDirect`, `ErrEmptyBody`, `ErrBodyTooLong`, `ErrNotGroup`, validación → 400 · resto → 500.

Eventos:

```go
const TopicMessageSent = "chat_room.message.sent" // payload MessageSentEvent: tenant_id, room_id, message_id, sender_id — NUNCA el body
const TopicGroupSaved  = "chat_room.group.saved"  // payload: la fila chat_room

const inboxTopicPrefix = "chat_room.inbox."

// InboxTopic es el canal de avisos de un usuario. La app suscribe la conexión push de cada
// usuario autenticado SOLO a su propio InboxTopic. El evento que se publica ahí no lleva payload:
// significa "algo cambió en tus salas, vuelve a pedir".
func InboxTopic(userID string) string { return inboxTopicPrefix + userID }
```

> Nota para quien revise: el id de usuario **dentro** del topic es deliberado. Es la dirección del
> buzón, no el tenant (que sigue yendo en el payload de `TopicMessageSent`). Así el canal push
> solo entrega a su dueño sin cambiar el contrato de `events.Event`.

---

## 4. Dependencias — `module.go`

```go
// ParticipantReader: los usuarios que pueden chatear. Key = user_id, Value = nombre visible.
type ParticipantReader interface {
	ParticipantOptions(tenantID string) ([]fmt.KeyValue, error)
}

type Deps struct {
	IDs           model.IDGenerator // requerido
	Participants  ParticipantReader // requerido
	TenantID      string            // requerido — tenant por defecto
	RetentionDays int               // requerido, > 0 — decisión de cada instalación, sin valor por defecto oculto
	Publisher     events.Publisher  // opcional — nil: no hay avisos push (la vista igual funciona al recargar)
	Clock         func() int64      // opcional — nil = time.Now (nanosegundos); inyectable en tests
}

func New(db *orm.DB, deps Deps) (*Module, error)
```

Errores de `New` (texto exacto): `chat_room: Deps.IDs is required`,
`chat_room: Deps.Participants is required`, `chat_room: Deps.TenantID is required`,
`chat_room: Deps.RetentionDays must be greater than zero`.

---

## 5. Reglas de dominio

Métodos exportados (el que llama va siempre como `userID`, que el handler toma de `ctx.UserID()`;
vacío → `ErrUnauthenticated`):

```go
ListParticipants(tenantID, userID string) ([]Participant, error)        // todos menos userID, con online
Heartbeat(tenantID, userID string) ([]Participant, error)               // upsert presencia; devuelve lo mismo que ListParticipants
OpenDirect(tenantID, userID, otherUserID string) (RoomSummary, error)   // get-or-create
ListRooms(tenantID, userID string) ([]RoomSummary, error)
ListMessages(tenantID, userID, roomID, afterID string, limit int) ([]MessageView, error)
SendMessage(tenantID, userID, roomID, body string) (MessageView, error)
MarkRead(tenantID, userID, roomID string) error
ListGroups(tenantID string) ([]Room, error)
SaveGroup(tenantID, id, name string) (Room, error)
SetGroupMembers(tenantID, roomID string, userIDs []string) error
ListGroupMembers(tenantID, roomID string) ([]Participant, error)
DeleteGroup(tenantID, id string) error                                  // en una Tx: mensajes, miembros, sala
PurgeExpired(tenantID string) (int, error)                              // borra mensajes con created_at < now - RetentionDays; devuelve cuántos
```

1. **Sala general.** `ensureBroadcast(tenantID)` (no exportada) crea la sala `broadcast` del
   tenant la primera vez que se necesita (`ListRooms`, `SendMessage`, `ListMessages`). Una sola por
   tenant.
2. **Membresía** — `isMember(tenantID, userID, room)`: `broadcast` → `userID` está en
   `ParticipantOptions`; `direct`/`group` → existe la fila `chat_member`. Toda op sobre una sala
   que no cumpla → `ErrNotMember`. **Ningún permiso RBAC salta esta regla.**
3. **Direct.** `direct_key = min(a,b) + "|" + max(a,b)` (comparación de cadenas). `OpenDirect`
   busca por `direct_key`; si no existe, crea la sala y **dos** filas `chat_member` en una `db.Tx`.
   `otherUserID` debe estar en `ParticipantOptions` (`ErrUnknownUser`) y ser distinto de `userID`
   (`ErrSelfDirect`).
4. **Enviar.** `body` sin espacios al borde; vacío → `ErrEmptyBody`; más de `MaxBodyLength`
   caracteres (runas, no bytes) → `ErrBodyTooLong`. `sender_label` = `Value` de `userID` en
   `ParticipantOptions`. Tras crear: `MarkRead` del emisor, publicar `TopicMessageSent` y un evento
   **sin payload** en `InboxTopic(u)` por **cada** destinatario `u` distinto del emisor (miembros de
   la sala; en `broadcast`, todos los participantes).
5. **No leídos.** `unread` = mensajes de la sala con `created_at > last_read_at` del usuario y
   `sender_id != userID`. En `broadcast` la fila `chat_member` del usuario se crea al primer
   `MarkRead`; sin fila, `last_read_at = 0`.
6. **Orden y paginación.** `ListMessages` ordena por `created_at` asc, luego `id` asc; `after_id` no
   vacío devuelve solo los posteriores a ese mensaje; `limit` acota desde el **final** (los más
   recientes).
7. **Grupos.** `SetGroupMembers` valida cada id contra `ParticipantOptions` (`ErrUnknownUser`) y
   reemplaza el conjunto en una Tx, **conservando** `last_read_at` de quienes siguen. Una sala que
   no es `group` → `ErrNotGroup`. Tras cambiar miembros, aviso a `InboxTopic` de cada miembro
   nuevo o quitado.
8. **En línea.** `online = now - last_seen < PresenceWindowSeconds` (convertir nanosegundos a
   segundos una vez, en una función no exportada).
9. **Retención.** `PurgeExpired` borra por `tenant_id` y `created_at`; **no** tiene op: la llama la
   app (ver `MASTER.md`).

---

## 6. Ops — `ops.go`

| Op | Gate | Accepts | Respuesta |
|---|---|---|---|
| `list_participants` | `.Authenticated()` | `ListRoomsArgs` | `ParticipantList` |
| `heartbeat` | `.Authenticated()` | `HeartbeatArgs` | `ParticipantList` |
| `open_direct` | `.Authenticated()` | `OpenDirectArgs` | `RoomSummary` |
| `list_rooms` | `.Authenticated()` | `ListRoomsArgs` | `RoomSummaryList` |
| `list_messages` | `.Authenticated()` | `ListMessagesArgs` | `MessageViewList` |
| `send_message` | `.Authenticated()` | `SendMessageArgs` | `MessageView` |
| `mark_read` | `.Authenticated()` | `MarkReadArgs` | 200 |
| `list_groups` | `.Requires(ResourceGroup, model.Read)` | `ListGroupsArgs` | `RoomList` |
| `save_group` | `.Requires(ResourceGroup, model.Create\|model.Update)` | `SaveGroupArgs` | `Room` |
| `set_group_members` | `.Requires(ResourceGroup, model.Update)` | `SetGroupMembersArgs` | 200 |
| `list_group_members` | `.Requires(ResourceGroup, model.Read)` | `ListGroupMembersArgs` | `ParticipantList` |
| `delete_group` | `.Requires(ResourceGroup, model.Delete)` | `DeleteGroupArgs` | 200 |

`.Authenticated()` es deliberado: chatear es una operación del usuario sobre sus propias salas; la
membresía (§5.2) es el control. Constantes `OpListParticipants = "list_participants"`, etc.
`tenant_id` vacío → `m.tenantID`.

---

## 7. `view.go`, `migrate/`, `seed/`

- `view.go`: `func (r *Room) Item() view.Item` (`Label: Name`) y `NewGroupView(caller)
  view.Presenter` con `Ops{Module: ModelName, List: OpListGroups, Save: OpSaveGroup, Delete:
  OpDeleteGroup}` — la administración de grupos es un CRUD normal (lo dibuja `crudview` en el
  segundo plan).
- `migrate/migrate.go`: `Sync` de `Room`, `Member`, `Message`, `Presence`, en ese orden.
- `seed/seed.go`: `Load(m *chatroom.Module, tenantID string, userIDs []string) (Data, error)` —
  un grupo "Recepción" con los dos primeros usuarios, un direct entre el primero y el segundo con 3
  mensajes, y 2 mensajes en General. Solo con métodos del módulo.

---

## 8. Tests — `tests/`

Archivos: `setup_test.go`, `new_test.go`, `direct_test.go`, `broadcast_test.go`, `group_test.go`,
`message_test.go`, `unread_test.go`, `presence_test.go`, `retention_test.go`, `inbox_test.go`,
`ops_test.go`, `tenant_test.go`, `seed_test.go`.

Casos obligatorios:

1. `New`: un error con el texto exacto por cada dependencia requerida.
2. `OpenDirect` dos veces (a→b y b→a) devuelve **la misma** sala; con uno mismo → `ErrSelfDirect`;
   con un desconocido → `ErrUnknownUser`.
3. Un tercero **no** puede `ListMessages`, `SendMessage` ni `MarkRead` en un direct ajeno →
   `ErrNotMember` (y 403 por la op). Un usuario con permiso `ResourceGroup` total tampoco.
4. `broadcast`: se crea una sola vez; todos los participantes pueden leer y enviar.
5. `SendMessage`: vacío, solo espacios y `MaxBodyLength+1` runas → error exacto; `sender_label`
   sale del lector, no del cliente.
6. No leídos: b recibe 2 mensajes → `unread = 2`; `MarkRead` → 0; los propios nunca cuentan.
7. `read` en direct: falso hasta que el otro llama `MarkRead`, luego verdadero; siempre falso en
   group/broadcast.
8. Paginación: `after_id` y `limit` según §5.6.
9. Presencia con `Clock` fijo: latido hace 89 s → en línea; hace 91 s → no.
10. `PurgeExpired`: borra solo los mensajes vencidos del tenant indicado; devuelve el conteo.
11. Buzón con `events/mock`: al enviar en un direct, exactamente **un** evento en
    `InboxTopic(b)` y ninguno en `InboxTopic(a)`; en broadcast con 3 participantes, dos; ningún
    evento publicado lleva el `body`.
12. `SetGroupMembers` conserva `last_read_at` de quien sigue.
13. Ops vía `router/mock`: sin `UserID` en el contexto → 401; gates `.Authenticated()` /
    `.Requires` según §6.
14. Aislamiento de tenant en todos los métodos.

---

## 9. Documentación

`docs/ARCHITECTURE.md` (español; incluye la tabla de decisiones de §0 y la nota sobre
`InboxTopic`), `docs/diagrams/database.md` (ERD de las 4 tablas), `README.md` (ops y archivos
clave), y las "Domain-specific notes" de `AGENTS.md`: genérico, sin pacientes, identidad solo
desde `ctx.UserID()`, `RetentionDays` obligatorio.

---

## 10. Etapas

| # | Etapa | Archivos | Criterio de aceptación |
|---|---|---|---|
| E1 | Modelo + esquema | borrar `chat_room.go`; `model.go`, `model_orm.go` (ormc), `migrate/` | `grep -rn "type ChatRoom struct" --include=*.go .` vacío; test de migrate verde |
| E2 | Dominio | `module.go`, `room.go`, `message.go`, `presence.go`, `group.go` | tests 1–12 y 14 verdes |
| E3 | Ops | `ops.go` | test 13 verde; `var _ router.OperationModule = (*Module)(nil)` |
| E4 | Vista de grupos + seed + docs | `view.go`, `seed/`, `docs/*`, `README.md` | `seed_test.go` verde; `grep -rn "TODO\|FIXME\|map\[" --include=*.go .` vacío; imports prohibidos ausentes; `gotest` verde en ambos carriles |
