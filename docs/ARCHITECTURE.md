# Arquitectura del Módulo `chat_room`

## 1. Visión general en un minuto

El módulo `chat_room` proporciona mensajería interna entre usuarios de una misma clínica o establecimiento (tenant) en el ecosistema Velty.

- **Salas flexibles**: Conversaciones directas 1:1, grupos de trabajo definidos por administración y una sala general de difusión comunitaria.
- **Persistencia desacoplada por `crud`**: El módulo no gestiona transacciones ni SQL directo en sus operaciones de entrada; la persistencia, paginación, estampas temporales y claves compuestas son gobernadas por el framework `crud` mediante definiciones declarativas en `definition.go`.
- **Reglas de negocio puras**: La lógica de negocio reside en archivos de reglas dedicados (`rule_*.go`) que validan la pertenencia, unicidad y permisos antes de persistir.
- **Privacidad estricta**: Los usuarios solo pueden ver y participar en salas donde son miembros explícitos. Intentar acceder a una sala ajena responde con «no encontrado» (`ErrNotFound`), ocultando su existencia.
- **Notificaciones push desacopladas**: Tras el envío de un mensaje se emite un aviso sin payload en el canal `InboxTopic(userID)` de cada destinatario; el contenido se recupera por lectura tipada.

## 2. Piezas y estructura

El módulo se compone de cinco tablas físicas y tres modelos de lectura virtual:

```
Tablas físicas:
  chat_room        Salas directas, grupos y difusión
  chat_member      Membresía por sala y usuario, con marca de último leído
  chat_message     Mensajes de texto persistidos con autoría y fecha
  chat_presence    Última conexión (latido) por usuario
  chat_direct      Pares ordenados de usuarios asociados a salas directas

Modelos de lectura (Wire):
  participant      Lista de usuarios del tenant con estado en línea
  room_summary     Resumen de salas con conteo de no leídos y última vista previa
  message_view     Vista detallada de mensajes con autoría propia y lectura
```

### Reglas vs Persistencia

El framework `crud` gobierna el ciclo de vida de los registros:
- Inyección de identificadores únicos (`m.ids.NewID()`) y partición por inquilino (`tenant_id`).
- Verificación de inmutabilidad en claves primarias y campos de solo servidor.
- Transaccionalidad atómica en operaciones por lote.
- Publicación de eventos de cambio (`crud.Topic`) tras la confirmación en base de datos.

Las reglas (`rule_*.go`) se concentran únicamente en invariantes de dominio: orden de participantes, pertenencia a la sala y longitud de texto.

## 3. Flujos paso a paso

### Envío de mensajes y `checkSendMessage`

Tanto la operación de creación `chat_message.create` como el método exportado `SendMessage` comparten la rutina `checkSendMessage(tx, roomID, senderID, body, m)`:
1. Valida que el texto no esté vacío y no exceda la longitud máxima (`MaxBodyLength`).
2. Verifica que el remitente sea miembro activo de la sala o que la sala sea de difusión comunitaria.
3. Asigna la estampa temporal actual y resuelve la etiqueta visible del remitente mediante `m.isParticipant`.
4. Emite avisos ligeros en `InboxTopic(destinatario)` tras la persistencia para notificar a los miembros conectados.

Compartir `checkSendMessage` garantiza que los límites de validación y la emisión de avisos se apliquen de forma idéntica sin importar el punto de entrada.

### Apertura de conversaciones directas (1:1)

1. El llamador invoca `chat_direct.create` con `{user_a, user_b}` ordenados lexicográficamente (`user_a < user_b`), siendo el emisor uno de los dos.
2. Si el par ya existía, la creación responde el registro existente de forma idempotente con su `room_id`.
3. Si es nuevo, la regla crea en una transacción atómica la sala `Room` de tipo directo, los dos registros `Member` correspondientes y la fila en `chat_direct`.

### Administración de miembros en grupos

Los miembros se modifican **un registro a la vez** (`chat_member.create` y `chat_member.delete`):
- No existe una operación monolítica de sincronización de miembros.
- Una asignación masiva se envía como un lote (`CreateArgs` con múltiples registros) ejecutado atómicamente por `crud`.
- El método exportado `SetGroupMembers` calcula internamente la diferencia entre miembros actuales y deseados, aplicando inserciones y eliminaciones puntuales.

### Latido de presencia

1. El cliente envía un latido mediante `chat_presence.update` sobre su propio identificador (o `create` en su primera conexión).
2. La regla verifica que el usuario autenticado solo actualice su propio registro (`old.UserId == a.UserID`).
3. Al consultar participantes (`participant.Read`), se consideran en línea aquellos cuyo último latido ocurrió dentro de la ventana de actividad (`HeartbeatWindowSeconds`).

## 4. Contratos y límites de seguridad

- **Aislamiento por inquilino**: `orm.TenantStore` asegura que ninguna consulta o modificación pueda alcanzar registros de otro `tenant_id`.
- **Límite de visibilidad («no encontrado»)**: Si un usuario intenta leer mensajes o acceder a una sala de la que no es miembro, el sistema responde `model.ErrorNotFound` en lugar de un error de autorización. Esto previene que un atacante deduzca la existencia o identificadores de salas privadas ajenas mediante enumeración.
- **Inmutabilidad de autoría y pertenencia**: `SenderId` y `SenderLabel` son determinados estrictamente a partir de la sesión autenticada (`a.UserID`), impidiendo la suplantación de identidad en los mensajes.
- **Actualización granular de miembros**: En `chat_member.update`, un participante únicamente tiene permitido modificar el campo `last_read_at` de su propio registro para actualizar confirmaciones de lectura; cualquier intento de alterar `room_id`, `user_id` o filas ajenas es rechazado.
