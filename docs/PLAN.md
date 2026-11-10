---
PLAN: "refactor!: chat_room on the crud-by-declaration framework — Allow and Rules in definition.go, persistence by crud, only domain rules in rule files"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 2956131194839037696
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Commit after every stage.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
> **Push early:** after EVERY stage run `git push origin HEAD`; after Stage 3 open the pull request and keep pushing to it.
> **Never ask a yes/no question about progress** («am I on the right track?», «shall I proceed?», «shall I push?»): the answer is always yes. Ask only when a compile error or a signature in this plan is wrong, and name the symbol.
> Never edit `definition_gen.go`. Never write `go.mod`/`go.sum` lines by hand: only `go get` and `go mod tidy` do. No scratch files (`*.py`, `*.sh`, no sed/regex scripts over the source): edit by hand, file by file, and run `gotest` after each file.

# Plan: `chat_room` on the crud-by-declaration framework

A first and a second session of this plan stalled on questions the plan did not answer. This version is split into small stages, one file each; **every answer they needed is written inside the stage that needs it**. Read the stage file, do it, `gotest`, commit, push, next.

Reference module, already migrated, the shape to copy: https://github.com/veltylabs/patient_directory/blob/main — `definition.go`, `rule_patient_create.go`, `rule_patient_update.go`, `rule_patient_search.go`, `store.go`, `import.go`, `module/server.go`, `seed/demo.go`, `tests/setup_test.go`, `tests/crud_test.go`, `tests/demo_test.go`.

## Stages

| # | File | What | Files touched |
|---|---|---|---|
| 1 | [PLAN_STAGE_1_TOOLS_DEFINITION.md](PLAN_STAGE_1_TOOLS_DEFINITION.md) | tools, `definition.go` | `definition.go`, `go.mod`, `go.sum`, `migrate/migrate.go` |
| 2 | [PLAN_STAGE_2_MODULE_STORE.md](PLAN_STAGE_2_MODULE_STORE.md) | `Module.store`, `store.go`, read helpers | `module.go`, `store.go`, `rows.go` |
| 3 | [PLAN_STAGE_3_RULES.md](PLAN_STAGE_3_RULES.md) | generate and write the rule files | `rule_*.go`, `checks.go` |
| 4 | [PLAN_STAGE_4_READERS.md](PLAN_STAGE_4_READERS.md) | readers and writers on `m.store`; delete the old paths | `group.go`, `message.go`, `room.go`, `presence.go`, `ops.go` (deleted) |
| 5 | [PLAN_STAGE_5_BROWSER.md](PLAN_STAGE_5_BROWSER.md) | `view.go`, `module/`, `seed/demo.go` | `view.go`, `module/*.go`, `seed/demo.go`, `web/` (deleted) |
| 6 | [PLAN_STAGE_6_TESTS.md](PLAN_STAGE_6_TESTS.md) | tests and `docs/TESTS.md` | `tests/*.go`, `docs/TESTS.md` |
| 7 | [PLAN_STAGE_7_DOCS.md](PLAN_STAGE_7_DOCS.md) | README, architecture, `langc` | `README.md`, `docs/ARCHITECTURE.md`, `lang.json` |

## Operation mapping (every operation of today has exactly one row)

`<entity>.<verb>` is `router.Operation{Module: ModelName, Entity: "<entity>", Action: model.<Verb>}`.

| Today | After |
|---|---|
| `list_participants`, `list_group_candidates` | read model `participant` |
| `heartbeat` | `chat_presence.create` (first time) / `chat_presence.update` with `["last_seen"]` |
| `open_direct` | `chat_direct.create` with `{user_a, user_b}` (sorted; the caller is one) |
| `list_rooms` | read model `room_summary` |
| `list_messages` | read model `message_view` with filter `{room_id, after_id}` |
| `send_message` | `chat_message.create` |
| `mark_read` | `chat_member.update` (id `room/user`) with `["last_read_at"]` |
| `list_groups`, `save_group`, `delete_group` | `chat_room.read` (filter `{kind:"group"}`), `.create`, `.update`, `.delete` |
| `set_group_members` | `chat_member.create` / `chat_member.delete`, one record per operation (a batch is a `CreateArgs` with several `Records`) |
| `list_group_members` | `chat_member.read` with filter `{room_id}` |

## Delivery check

1. `gotest` ends green on EVERY destination (`wasm` and `browser` included) and «tests index» ✅. A red destination is a failed delivery.
2. `langc sync .` reports zero untranslated keys for `es` and zero reuse warnings.
3. `grep -rn "reg.Operation\|view.MountSave\|view.MountDelete\|router.Context\|rpc.WriteError\|m\.db\." --include=*.go . | grep -v "_test.go\|^./ui/\|^./definition_gen.go"` → only `m.db` uses inside `module.go`'s constructor (the store is built from it) and `store.go`.
4. `ls *.py *.sh commit_message.txt 2>/dev/null` → empty; `grep -c XXXX go.sum` → 0.
5. `git diff --stat main` lists only the files of the stages table plus badge files `gotest` rewrites; nothing outside it is deleted.
