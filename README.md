# chat_room
<img src="docs/img/badges.svg">

Módulo de mensajería interna entre usuarios de un establecimiento (conversaciones directas 1:1, grupos de trabajo y sala general de difusión). Persistencia declarativa mediante [`crud`](https://github.com/webtyp/crud) y reglas de negocio puras. Más detalles en [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## ¿Por qué existe?

Proporciona comunicación en tiempo real entre el personal clínico con salas directas, grupos y difusión por establecimiento.

- **Sin esto:** Cada aplicación inventa protocolos ad-hoc de mensajería, consultas SQL dispersas, sincronización manual de presencia y filtrado manual de membresía.
- **Con esto:** Salas seguras gobernadas por reglas CRUD declarativas, pertenencia obligatoria para lectura y avisos push desacoplados del contenido.

### Operaciones (Antes vs Después)

| Antes | Después |
|---|---|
| `list_participants`, `list_group_candidates` | modelo de lectura `participant` |
| `heartbeat` | `chat_presence.create` / `chat_presence.update` con `["last_seen"]` |
| `open_direct` | `chat_direct.create` con `{user_a, user_b}` ordenados |
| `list_rooms` | modelo de lectura `room_summary` |
| `list_messages` | modelo de lectura `message_view` con filtro `{room_id, after_id}` |
| `send_message` | `chat_message.create` |
| `mark_read` | `chat_member.update` (id `room/user`) con `["last_read_at"]` |
| `list_groups`, `save_group`, `delete_group` | `chat_room.read` (filtro `{kind:"group"}`), `.create`, `.update`, `.delete` |
| `set_group_members` | `chat_member.create` / `chat_member.delete` por registro |
| `list_group_members` | `chat_member.read` con filtro `{room_id}` |

### Permisos

| Recurso | Acciones | Asignación |
|---|---|---|
| `chat_message` | `Create` | `DefaultGrants` (todo usuario autenticado) |
| `chat_presence` | `Create`, `Update` | `DefaultGrants` (solo su propio registro) |
| `chat_direct` | `Create`, `Read` | `DefaultGrants` (si es participante del par) |
| `chat_member` | `Read`, `Update` | `DefaultGrants` (`Update` solo `last_read_at` propio) |
| `chat_room` | `Read` | `DefaultGrants` (salas donde es miembro y difusión) |
| `chat_room` / `chat_member` | `Create`, `Delete` | Rol administrador de la aplicación |

## Antes de comenzar

- Entorno Go configurado con `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`.
- Una base de datos `*orm.DB` con las tablas de [migrate/migrate.go](migrate/migrate.go) aplicadas.
- Dependencias obligatorias en `chatroom.Deps`: `IDs`, `Participants`, `TenantID` y `RetentionDays` (> 0).

## Guía paso a paso

### Paso 1: Inicializar el módulo

```go
package main

import (
	chatroom "github.com/veltylabs/chat_room"
	"webtyp.com/fmt"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
	"webtyp.com/unixid"
)

type fakes struct{}

func (fakes) ParticipantOptions(tenantID string) ([]fmt.KeyValue, error) {
	return []fmt.KeyValue{{Key: "u1", Value: "Ana"}, {Key: "u2", Value: "Bruno"}}, nil
}

func main() {
	ids, err := unixid.NewUnixID()
	if err != nil { fmt.Println(err); return }
	db := orm.New(mem.New())
	mod, err := chatroom.New(db, chatroom.Deps{
		IDs: ids, Participants: fakes{}, TenantID: "t1", RetentionDays: 30,
	})
	if err != nil { fmt.Println(err); return }
	fmt.Println(mod.ModelName()) // chat_room
}
```

### Paso 2: Enviar mensajes y operar

Las llamadas cliente se realizan vía `crud.LoopbackActingAs` o transporte HTTP usando operaciones tipadas. Ver catálogo completo en [tests/crud_test.go](tests/crud_test.go).

### Paso 3: Ejecutar la demostración

Ejecuta el entorno interactivo en la raíz del repositorio:
```bash
webtyp
```
El punto de entrada de la demo se encuentra en [seed/demo.go](seed/demo.go).

## Mapa de documentación

| Orden | Documento | Qué pregunta responde y cuándo leerlo |
|---|---|---|
| 1 | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | ¿Cómo interactúan las reglas de negocio, la persistencia y los límites de seguridad? |
| 2 | [docs/TESTS.md](docs/TESTS.md) | ¿Qué pruebas existen y qué regla de negocio garantiza cada una? |
| 3 | [docs/diagrams/database.md](docs/diagrams/database.md) | ¿Cuál es el modelo relacional y cómo se vinculan las tablas? |
