← [Stage 6](PLAN_STAGE_6_TESTS.md)

# Stage 7 — translations and documentation (Spanish, `veltylabs`)

1. `langc sync .`: zero pending keys for `es` and zero reuse warnings (other languages being empty in `lang.json` is a warning of the repo, not of this plan). Add `es` text for any new key (`ErrNameInvalid`, `ErrFieldNotEditable`, `ErrRoomRequired`, `ErrDirectPairInvalid` are constants, not `lang.Text`: they need no key).
2. `README.md` (≤150 lines), four sections in this order, all in Spanish:
   - `## ¿Por qué existe?`: what the module is (a chat for the people of a clinic: direct conversations, groups and a general room) and when the reader meets it; **Sin esto vs. Con esto**; first mentions linked ([`crud`](https://github.com/webtyp/crud), [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)).
   - `## Antes de comenzar`: Go and `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`; a `*orm.DB` with the tables of [migrate](migrate/migrate.go); `Deps.IDs`, `Deps.Participants`, `Deps.TenantID`, `Deps.RetentionDays`.
   - `## Guía paso a paso`: Step 1 create the module over `orm.New(mem.New())` (a complete `main` importing `"webtyp.com/fmt"`, never stdlib `fmt`, with `fmt.Println(err); return` on error, a `ParticipantReader` double returning `u1` and `u2`, and printing `mod.ModelName()` → `chat_room`); Step 2 send a message (link [tests/crud_test.go](tests/crud_test.go) for the complete operations); Step 3 open the demo (`webtyp` at the repo root, [seed/demo.go](seed/demo.go)).
   - `## Mapa de documentación`: table `Orden | Documento | Qué pregunta responde y cuándo leerlo` with `docs/ARCHITECTURE.md`, `docs/TESTS.md` and each file of `docs/diagrams/`.
   Include the **operations table** («Antes» / «Después») of the mapping in `PLAN.md`, a permissions table (resource, actions, who: `DefaultGrants` for every signed-in person; `chat_room` and `chat_member` create/delete only for the administrator role) and the «Sin/Con» comparison.
3. `docs/ARCHITECTURE.md`: four sections (`## 1. Visión general en un minuto`, `## 2. Piezas y estructura`, `## 3. Flujos paso a paso`, `## 4. Contratos y límites de seguridad`): rules vs persistence, why `SendMessage` and the rule share `checkSendMessage`, why a person outside a room gets «not found», that members change one record at a time.
4. Translate to Spanish every README, `docs/*`, diagram and code comment still in English; every exported symbol without a usage example gets one.
5. Permanent docs never link `docs/PLAN.md` or its stage files.

Run `gotest` and `langc sync .` one last time and read their output, then check the **Delivery check** of `PLAN.md`, commit, push and make sure the pull request is open.
