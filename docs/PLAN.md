---
PLAN: "fix: detect sentinel errors without == between interfaces (no reflection in wasm)"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 9316579022149767910
---

# Plan — `chat_room`: errores centinela sin `==` entre interfaces

> Master: `webtyp/docs/NO_REFLECTION_MASTER_PLAN.md` (ola 3). Doctrina: skill `api-design`.
> **Prerrequisito:** `go get webtyp.com/orm@latest` y confirmar que existe `orm.IsNotFound`. Si falta alguna, parar y reportarlo: no implementar un sustituto local.

## 1. El problema

En TinyGo, `==`, `!=` y `switch` entre valores de interfaz compilan a `runtime.interfaceEqual`, que
llama a `reflectValueEqual(reflectlite.ValueOf(x), reflectlite.ValueOf(y))`. `error` es una interfaz:
cada `err == ErrX` mete `internal/reflectlite` (~9 KB) en el binario wasm. La regla del dueño es que
el código que compila a wasm no use reflexión nunca. `errors.Is`/`errors.As` tampoco sirven: también
usan reflectlite.

## 2. La corrección — dos patrones, ninguno más

**A. Centinelas de otros paquetes** — usar su función de consulta:

| Antes | Después |
|---|---|
| `err == orm.ErrNotFound` | `orm.IsNotFound(err)` |
| `err != orm.ErrNotFound` | `!orm.IsNotFound(err)` |
| `err == storage.ErrNoRows` | `storage.IsNoRows(err)` |

**B. Centinelas propios de este paquete** — un tipo string no exportado; se afirma una vez y se
compara el valor concreto (comparación de strings, sin reflexión):

```go
// domainError is the concrete type of this package's sentinel errors. Code
// compares them by asserting this type and comparing the value: == between two
// error values compiles, under TinyGo, to runtime.interfaceEqual, which pulls
// internal/reflectlite into the wasm binary.
type domainError string

func (e domainError) Error() string { return string(e) }

const (
	ErrNotFound domainError = "<texto actual>"
	// … uno por centinela, con su texto actual
)
```

- `<texto actual>`: el string exacto que devuelve hoy el centinela (`fmt.Err("a", "b")` une las
  palabras con un espacio: `"a b"`). Un test fija cada texto: los mensajes no cambian.
- Uso, por ejemplo al traducir errores a códigos:

```go
if e, ok := err.(domainError); ok {
	switch e {
	case ErrFloorInUse, ErrRoomOverlap:
		return conflict
	case ErrNotFound:
		return notFound
	}
}
if orm.IsNotFound(err) {
	return notFound
}
```

- Un `switch err { case ErrA: … }` pasa a `if e, ok := err.(domainError); ok { switch e { … } }`.
- Si un centinela propio se envuelve antes de compararlo (`fmt.Errf("…%v", ErrX)`), la comparación
  con `==` ya no funcionaba: dejarlo igual y anotarlo en el PR, no inventar otra detección.

## 3. Sitios a cambiar (inventario del 2026-10-08)

### Código de producción

- `message.go:55` — `if err == orm.ErrNotFound {`
- `message.go:77` — `if err == orm.ErrNotFound {`
- `message.go:170` — `if err == orm.ErrNotFound {`
- `message.go:289` — `if err == orm.ErrNotFound {`
- `message.go:322` — `if err == orm.ErrNotFound {`
- `ops.go:81` — `switch err {`
- `ops.go:82` — `case ErrUnauthenticated:`
- `ops.go:84` — `case ErrNotMember:`
- `ops.go:86` — `case ErrNotFound:`
- `ops.go:88` — `case ErrTenantRequired, ErrUnknownUser, ErrSelfDirect, ErrEmptyBody, ErrBodyTooLong, ErrNotGroup:`
- `presence.go:70` — `} else if err == orm.ErrNotFound {`
- `room.go:96` — `if err != nil && err != orm.ErrNotFound {`
- `room.go:100` — `if err == orm.ErrNotFound {`
- `room.go:252` — `} else if err != orm.ErrNotFound {`
- `room.go:298` — `if err == orm.ErrNotFound {`
- `group.go:46` — `if err == orm.ErrNotFound {`
- `group.go:110` — `if err == orm.ErrNotFound {`
- `group.go:270` — `if err == orm.ErrNotFound {`
- `group.go:345` — `if err == orm.ErrNotFound {`

### Centinelas propios de este repo (patrón B)

- `model.go:223` — `ErrNotFound        = fmt.Err("chat_room: not found")`
- `model.go:224` — `ErrTenantRequired  = fmt.Err("chat_room: tenant_id is required")`
- `model.go:225` — `ErrUnauthenticated = fmt.Err("chat_room: authenticated user required")`
- `model.go:226` — `ErrNotMember       = fmt.Err("chat_room: not a member of this room")`
- `model.go:227` — `ErrUnknownUser     = fmt.Err("chat_room: unknown participant")`
- `model.go:228` — `ErrSelfDirect      = fmt.Err("chat_room: cannot open a direct room with yourself")`
- `model.go:229` — `ErrEmptyBody       = fmt.Err("chat_room: message body is empty")`
- `model.go:230` — `ErrBodyTooLong     = fmt.Err("chat_room: message body is too long")`
- `model.go:231` — `ErrNotGroup        = fmt.Err("chat_room: room is not a group")`

### Tests (se migran igual: un solo camino también en los tests)

- `tests/message_test.go:27` — `if err != chatroom.ErrEmptyBody {`
- `tests/message_test.go:33` — `if err != chatroom.ErrEmptyBody {`
- `tests/message_test.go:43` — `if err != chatroom.ErrBodyTooLong {`
- `tests/extra_branches_test.go:25` — `if err != chatroom.ErrUnauthenticated {`
- `tests/extra_branches_test.go:30` — `if err != chatroom.ErrUnauthenticated {`
- `tests/extra_branches_test.go:35` — `if err != chatroom.ErrUnauthenticated {`
- `tests/extra_branches_test.go:40` — `if err != chatroom.ErrUnauthenticated {`
- `tests/extra_branches_test.go:45` — `if err != chatroom.ErrUnauthenticated {`
- `tests/extra_branches_test.go:50` — `if err != chatroom.ErrUnauthenticated {`
- `tests/extra_branches_test.go:55` — `if err != chatroom.ErrUnauthenticated {`
- `tests/extra_branches_test.go:61` — `if err != chatroom.ErrNotFound {`
- `tests/extra_branches_test.go:66` — `if err != chatroom.ErrNotFound {`
- `tests/extra_branches_test.go:71` — `if err != chatroom.ErrNotFound {`
- `tests/extra_branches_test.go:76` — `if err != chatroom.ErrNotFound {`
- `tests/extra_branches_test.go:87` — `if err != chatroom.ErrNotGroup {`
- `tests/extra_branches_test.go:92` — `if err != chatroom.ErrNotGroup {`
- `tests/extra_branches_test.go:97` — `if err != chatroom.ErrNotGroup {`
- `tests/extra_branches_test.go:102` — `if err != chatroom.ErrNotGroup {`
- `tests/extra_branches_test.go:108` — `if err != chatroom.ErrUnknownUser {`
- `tests/extra_branches_test.go:114` — `if err != chatroom.ErrNotFound {`
- `tests/extra_branches_test.go:125` — `if err != chatroom.ErrUnknownUser {`
- `tests/extra_branches_test.go:131` — `if err != chatroom.ErrNotFound {`
- `tests/extra_branches_test.go:136` — `if err != chatroom.ErrNotFound {`
- `tests/extra_branches_test.go:141` — `if err != chatroom.ErrNotFound {`
- `tests/extra_branches_test.go:147` — `if err != chatroom.ErrNotFound {`
- `tests/direct_test.go:38` — `if err != chatroom.ErrSelfDirect {`
- `tests/direct_test.go:44` — `if err != chatroom.ErrUnknownUser {`
- `tests/direct_test.go:72` — `if err != chatroom.ErrNotMember {`
- `tests/direct_test.go:78` — `if err != chatroom.ErrNotMember {`
- `tests/direct_test.go:84` — `if err != chatroom.ErrNotMember {`

Si encuentras otro `==`/`!=`/`switch` entre valores de interfaz con operandos no nil que no esté en la
lista, se migra igual. `x == nil` y `x != nil` están bien.

## 4. Tests

- Todos los tests existentes siguen verdes sin cambiar su intención.
- Un test que fija el `Error()` de cada centinela propio convertido (patrón B) contra su texto anterior.
- Si el paquete traduce errores a códigos/respuestas (por ejemplo en `ops.go`), un test por rama
  cambiada: el mismo error produce el mismo código que antes.
- `gotest` verde (vet, race, tests, wasm).

## 5. Criterios de aceptación

- `grep -rnE '(==|!=) *[A-Za-z_.]*Err[A-Za-z]*' --include=*.go . | grep -v '_temp/'` → vacío.
- `grep -rn 'switch err {' --include=*.go .` → vacío.
- `grep -rn 'errors.Is\|errors.As' --include=*.go .` → vacío.
- Ningún símbolo exportado nuevo: `git diff | grep '^+func [A-Z]'`.
- `gotest` verde.

## 6. Restricciones

Las de `AGENTS.md`, más: nada de `reflect`, `unsafe`, `errors.Is`/`errors.As`, ni `==`/`!=`/`switch`
entre valores de interfaz con operandos no nil. No tocar otros repos.

## Executor notes
- Added `domainError` in `model.go` to replace interface-based errors, to avoid pulling reflectlite into the wasm binary.
- Updated all occurrences of `err == orm.ErrNotFound` and `err != orm.ErrNotFound` to use `orm.IsNotFound(err)` instead.
- Refactored `switch err { ... }` in `ops.go` to use type assertion first (`if e, ok := err.(domainError); ok { switch e { ... } }`).
- Updated the assertions in test files to use `err == nil || err.Error() != chatroom.ErrX.Error()` since `domainError` is unexported and `tests` is an external test package, which restricts the ability to use `err.(domainError)`.
- Replaced `fmt.Err` usage in definitions of sentinel errors.
- Added `TestSentinelErrors` to guarantee the underlying sentinel error messages match exactly the text prior to changes.
