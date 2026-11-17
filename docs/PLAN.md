---
PLAN: "refactor: chat_room rules answer with crud.RuleAnswer, Search stops searching text"
EXECUTOR: antigravity
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work.

# Plan — `github.com/veltylabs/chat_room`: bring the module to the current `crud`

Orchestrated from the owner's `EXECUTION_ORDER_MASTER_PLAN` (block E). **Do not start before the versions table below has no `{{` left** (it is filled when the framework release of «RECORD_SEARCH phase 2» is published).

## 0. Context (verified in a scratch copy on 2026-11-16: every command below was run and `gotest` ended green on every destination — do not re-diagnose)

`chat_room` still pins an old `webtyp.com/crud` (v0.0.11/v0.0.12). Three framework changes reached it:

1. **`crud.RuleAnswer` is a struct, and `ApplyAndPublish` is a function.** A rule that rejects returns `crud.RuleAnswer{}` (the zero value rejects); a rule that accepts returns `crud.ApplyAndPublish()`. `return 0, err` and `return crud.ApplyAndPublish, nil` no longer compile.
2. **A `Search` rule no longer receives the typed text.** `crud` itself filters every typed word on the hidden column `text_search` (accent- and case-insensitive); the rule only adds AND conditions (state, site…). The parameter `text string` is removed from the signature, and any branch that searched the text with `LIKE` is deleted.

Anti-footguns: do **not** touch `definition_gen.go` by hand (a later stage regenerates it); do **not** add a `replace` to `go.mod`; do **not** write `go.sum` lines (only `go get` and `go mod tidy` write them); do **not** edit with `sed`/regex/scripts — one file at a time with the editor, `gotest` at the end of each stage; do **not** change any rule's logic, only what this plan lists.

## 1. Tools and versions

Versions (filled at dispatch from the published release; a `{{` left means: stop and say so in the PR):

| Module | Tag |
|---|---|
| `webtyp.com/model` | `v0.2.24` |
| `webtyp.com/input` | `v0.2.5` |
| `webtyp.com/storage` | `v0.2.8` |
| `webtyp.com/orm` | `v0.12.42` |
| `webtyp.com/ddl` | `v0.0.57` |
| `webtyp.com/crud` | `v0.0.17` |
| `webtyp.com/definitioncheck` | `v0.0.10` |
| `webtyp.com/definitionc` | `v0.1.49` |
| `webtyp.com/view` | `v0.6.86` |

```bash
export GOPRIVATE="webtyp.com/*,github.com/veltylabs/*" GOFLAGS=-mod=mod
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/definitionc/cmd/definitionc@v0.1.49
export PATH="$PATH:$(go env GOPATH)/bin"
go get webtyp.com/model@v0.2.24 webtyp.com/input@v0.2.5 webtyp.com/storage@v0.2.8 webtyp.com/orm@v0.12.42 webtyp.com/ddl@v0.0.57 webtyp.com/crud@v0.0.17 webtyp.com/definitioncheck@v0.0.10 webtyp.com/view@v0.6.86
go mod tidy
```

Run `go vet ./...` once: the errors it prints are exactly the sites of stages 3–4 (and `definition_gen.go`, which stage 5 regenerates). `go.mod` must end with no `replace` line.

## 2. APIs used (read from the published tags above; if a signature differs, stop and name the symbol in the PR — never adapt it by guessing)

| Symbol | Exact signature | File |
|---|---|---|
| `crud.RuleAnswer` | `type RuleAnswer struct{…}`; its zero value is a rejection | `crud/result.go` |
| `crud.ApplyAndPublish` | `func ApplyAndPublish(calls ...ExternalCall) RuleAnswer` | `crud/result.go` |
| `crud.ApplyWithoutPublishing` | `func ApplyWithoutPublishing(calls ...ExternalCall) RuleAnswer` | `crud/result.go` |
| `Search` hook the generator wires | `func(a crud.CurrentUser, filter model.Model) ([]storage.Condition, error)`; your method is `func (p *X) Search(a crud.CurrentUser, filter *X, m *Module) ([]storage.Condition, error)` | `crud/contract.go` |

## 3. Stage — the rules answer with `crud.RuleAnswer` (one file at a time; `gotest` after the last)

In each file below, inside the rule's body only: every `return 0, <err>` becomes `return crud.RuleAnswer{}, <err>`, and the final `return crud.ApplyAndPublish, nil` becomes `return crud.ApplyAndPublish(), nil` (or `crud.ApplyWithoutPublishing` → `crud.ApplyWithoutPublishing()` where the file has it). Nothing else changes. Worked example (`rule_Post_update.go` of `editorial`):

```go
// before
return 0, &model.FieldError{Field: "state", Err: ErrInvalidTransition}
return crud.ApplyAndPublish, nil
// after
return crud.RuleAnswer{}, &model.FieldError{Field: "state", Err: ErrInvalidTransition}
return crud.ApplyAndPublish(), nil
```

| File | `return 0,` to change | `crud.Apply…` to call |
|---|---|---|
| `rule_chat_direct_create.go` | 6 | 1 |
| `rule_chat_member_create.go` | 4 | 1 |
| `rule_chat_member_delete.go` | 2 | 1 |
| `rule_chat_member_update.go` | 2 | 1 |
| `rule_chat_message_create.go` | 1 | 1 |
| `rule_chat_presence_create.go` | 3 | 1 |
| `rule_chat_presence_update.go` | 2 | 1 |
| `rule_chat_room_create.go` | 1 | 1 |
| `rule_chat_room_delete.go` | 2 | 1 |
| `rule_chat_room_update.go` | 3 | 1 |

Total: 26 returns and 10 calls. Check: `grep -n "return 0, " rule_*.go` → empty; `grep -nE "crud\.Apply(AndPublish|WithoutPublishing)," rule_*.go` → empty.

## 4. Stage — `Search` rules lose `text`

| File | New signature (line 1 of the function) | Also delete |
|---|---|---|
| `rule_chat_member_search.go` | `func (p *Member) Search(a crud.CurrentUser, filter *Member, m *Module) ([]storage.Condition, error) {` | — |
| `rule_chat_room_search.go` | `func (p *Room) Search(a crud.CurrentUser, filter *Room, m *Module) ([]storage.Condition, error) {` | the `if text != ""` block (3 lines) and its `storage.Like`/`orm.Like` call |

Never replace a deleted text branch with another text search: `crud` already searches `text_search`. Example (`chat_room`, `rule_chat_room_search.go`):

```go
// before
func (p *Room) Search(a crud.CurrentUser, filter *Room, text string, m *Module) ([]storage.Condition, error) {
	var conditions []storage.Condition
	if filter != nil && filter.Kind != "" {
		conditions = append(conditions, storage.Eq(Room_.Kind, filter.Kind))
	}
	if text != "" {
		conditions = append(conditions, storage.Like(Room_.Name, "%"+text+"%"))
	}
	return conditions, nil
}
// after
func (p *Room) Search(a crud.CurrentUser, filter *Room, m *Module) ([]storage.Condition, error) {
	var conditions []storage.Condition
	if filter != nil && filter.Kind != "" {
		conditions = append(conditions, storage.Eq(Room_.Kind, filter.Kind))
	}
	return conditions, nil
}
```

Check: `grep -n "text string" rule_*search.go` → empty.

## 5. Stage — regenerate `definition_gen.go`

```bash
definitionc
go get webtyp.com/model@v0.2.24 webtyp.com/input@v0.2.5 webtyp.com/storage@v0.2.8 webtyp.com/orm@v0.12.42 webtyp.com/ddl@v0.0.57 webtyp.com/crud@v0.0.17 webtyp.com/definitioncheck@v0.0.10 webtyp.com/view@v0.6.86
go mod tidy
```

`definitionc` rewrites `go.mod` to older versions: the `go get` line and `go mod tidy` after it are mandatory. The regenerated `Search:` hooks lose `text string`. Never edit `definition_gen.go`. Run `gotest`.

**Open the pull request now** (after this stage) and keep committing to it; commit after every stage; never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.

## 6. Stage — documentation and language

- `docs/TESTS.md`: unchanged unless a test name changed (none does); `gotest` checks the index.
- `grep -rn "ApplyAndPublish\b[^(]\|return 0," README.md docs/*.md docs/diagrams/*.md` → fix any snippet that still shows the old form with the new one.
- Language (veltylabs): translate to Spanish every English comment in the files this plan touched (the rule files above); do not rewrite what is already Spanish.

## 7. Delivery check

1. `gotest` ends green on **every destination it lists** (`vet`, `style`, `race`, `tests`, `wasm`, `browser`, `tests index`). A PR that does not compile or whose tests could not run is a failed delivery.
2. `grep -n "replace" go.mod` → empty; `go.sum` written only by `go get`/`go mod tidy`.
3. The greps of stages 3–4 return empty; after the regeneration `grep -n "text string" definition_gen.go` → empty too.
4. When it passes: commit, push, open the pull request. Stuck for 30 minutes: open it anyway, listing what fails.

## Stages

| # | Stage | Done when |
|---|---|---|
| 1 | Tools and versions | `go vet ./...` prints only the sites of stages 3–4 |
| 3 | `crud.RuleAnswer` | the two greps of stage 3 are empty |
| 4 | `Search` without `text` | `grep "text string" rule_*search.go` empty |
| 5 | `definitionc` | `gotest` green; PR open |
| 6 | docs and language | greps empty |
