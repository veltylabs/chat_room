# Tests

| Test | Rule |
|---|---|
| [TestBroadcastRoom](../tests/broadcast_test.go) | La sala general de difusión se crea automáticamente por tenant y permite enviar y leer mensajes a todos los usuarios |
| [TestBroadcastRoom_ListedOnceAfterPosting](../tests/broadcast_test.go) | Enviar mensajes a la sala general no duplica su aparición en la lista de salas |
| [TestMessage_NonMemberRejected](../tests/crud_test.go) | Un usuario que no pertenece al grupo no puede enviar mensajes a la sala |
| [TestMessage_CreateSetsSenderFields](../tests/crud_test.go) | La creación de mensaje establece automáticamente el remitente autenticado y marca la autoría propia |
| [TestMessages_NonMemberCannotRead](../tests/crud_test.go) | Un usuario ajeno a una sala recibe `ErrorNotFound` al intentar leer sus mensajes mediante el modelo de lectura |
| [TestMessages_ReadModelNeedsRoom](../tests/crud_test.go) | La lectura de mensajes mediante el modelo `message_view` exige especificar obligatoriamente `room_id` |
| [TestMember_MarkReadOnlyOwnRow](../tests/crud_test.go) | Un miembro solo puede actualizar la marca de lectura de su propio registro de membresía |
| [TestMember_UpdateOnlyLastRead](../tests/crud_test.go) | La actualización de membresía solo permite modificar `last_read_at` y rechaza alterar otros campos |
| [TestMember_CreateNeedsKnownGroupUser](../tests/crud_test.go) | Agregar un miembro a un grupo requiere un usuario participante conocido del tenant |
| [TestDirect_SamePairOnce](../tests/crud_test.go) | La creación de chat directo entre dos usuarios es idempotente y reutiliza la misma sala existente |
| [TestDirect_CallerMustBeInPair](../tests/crud_test.go) | El usuario que inicia un chat directo debe ser uno de los dos participantes del par |
| [TestDirect_PairMustBeSorted](../tests/crud_test.go) | La creación en `chat_direct` exige que el par de identificadores esté ordenado lexicográficamente |
| [TestPresence_OnlyOwn](../tests/crud_test.go) | Un usuario solo puede actualizar su propio registro de presencia |
| [TestGroup_DeleteCascades](../tests/crud_test.go) | Eliminar un grupo remueve en cascada sus miembros y mensajes asociados |
| [TestGroup_DeleteRejectsNonGroup](../tests/crud_test.go) | Solo las salas de tipo grupo pueden eliminarse, rechazando la eliminación de salas de difusión |
| [TestMount_MatchesDefinition](../tests/crud_test.go) | Las operaciones montadas en el registro coinciden exactamente con la definición del módulo |
| [TestCreate_PublishesChange](../tests/crud_test.go) | La creación de una sala publica el evento de cambio correspondiente a través del publicador |
| [TestDefaultGrants_SignedInCanSend](../tests/crud_test.go) | Los permisos por defecto otorgan creación de mensajes a usuarios autenticados sin permisos de eliminación |
| [TestDemo_Valid](../tests/demo_test.go) | La función `seed.Demo` genera una demostración ejecutable válida sobre WASM con pantallas y módulos |
| [TestOpenDirect](../tests/direct_test.go) | Abrir una conversación directa crea la sala y asocia a ambos participantes |
| [TestDirectPrivacy](../tests/direct_test.go) | Una conversación directa solo es visible y accesible para sus dos participantes |
| [TestExtraValidationAndErrorBranches](../tests/extra_branches_test.go) | Validación de nombres de grupo vacíos, usuarios no autenticados y parámetros fuera de rango |
| [TestListGroupCandidates](../tests/group_candidates_test.go) | La lista de candidatos para grupos incluye todos los participantes del tenant |
| [TestGroupManagement](../tests/group_test.go) | Creación, actualización de miembros, listado y eliminación de grupos |
| [TestInboxEvents](../tests/inbox_test.go) | El envío de mensajes publica notificaciones en el buzón personal de cada destinatario |
| [TestSendMessageValidation](../tests/message_test.go) | Validación de longitud máxima y contenido no vacío al enviar mensajes |
| [TestMessagePagination](../tests/message_test.go) | Paginación secuencial de mensajes mediante el identificador cursor `after_id` |
| [TestNewDependenciesValidation](../tests/new_test.go) | La inicialización del módulo valida que las dependencias obligatorias no sean nulas |
| [TestPresence](../tests/presence_test.go) | El latido de presencia actualiza el estado en línea dentro de la ventana de tiempo configurada |
| [TestPurgeExpired](../tests/retention_test.go) | La purga por retención elimina mensajes anteriores a la ventana de días configurada |
| [TestScreen_SaveAndDeleteGroup](../tests/screen_save_test.go) | La interfaz de pantalla permite guardar y eliminar grupos mediante operaciones CRUD |
| [TestSentinelErrors](../tests/sentinels_test.go) | Los errores centinela del paquete exponen textos descriptivos e inmutables |
| [TestChatScreenLiveWasm](../tests/ui_live_browser_test.go) | Renderizado interactivo en navegador WASM del chat, contadores de no leídos y administración de miembros |
| [TestUnreadAndReadReceipts](../tests/unread_test.go) | Cálculo de mensajes no leídos y actualización de confirmaciones de lectura |
| [TestClockAndShortTime](../ui/time_internal_test.go) | Formateo horario y etiquetas relativas de tiempo según la zona de la instalación |
