---
PLAN: "feat: the demo shows its API and its database (layout v0.3.99 developer tools)"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 9944839386845840309
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Anything unclear is an entry
> under "## Executor notes" in that file and in the pull request.

# Developer tools in the demo and translation to Spanish

## Context

This repo's demo (`web/browser.go`) runs entirely in the browser: its operations go through `loopback` and its data lives
in memory, with no server. Until now the menu always added «Developer tools», and both screens asked a server that
does not exist for `/_routes` and `/_schema` (404). `webtyp.com/layout` v0.3.99 fixes that: `BuildMenu` now takes the
tools as a **required** fourth argument, and `app.LocalTools(modules, steps...)` builds them in the browser with the
same pieces the server uses — the routes the operations declare and the migration steps. So this demo stops compiling
until it passes its tools; this plan does exactly that. In a demo the explorer shows the table without the «try» form
(there is no server to send a request to).

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go get webtyp.com/layout@v0.3.99
go mod tidy
```

Environment: `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, `GOFLAGS=-mod=mod`.

## APIs used

| Symbol | Exact signature | Tag |
|---|---|---|
| `app.LocalTools` | `func LocalTools(modules []router.OperationModule, steps ...ddl.Step) (Tools, error)` | `webtyp.com/layout/app` v0.3.99 |
| `app.BuildMenu` | `func BuildMenu(screens []view.Screen, ids model.IDGenerator, allows func(model.Resource) bool, tools Tools) (Menu, error)` | v0.3.99 |
| `ddl.Step` | `struct { Name string; Run func(conn Execer, compiler Compiler) error }` | `webtyp.com/ddl` |
| `migrate.Migrate` | `func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error` — fits `ddl.Step.Run` | `github.com/veltylabs/chat_room/migrate` |

In `web/browser.go` and `tests/ui_live_browser_test.go` the layout package is imported as `app`.

## Stage 1 — `web/browser.go` and `tests/ui_live_browser_test.go`

1. In `web/browser.go`, add the imports:
```go
	chatroommigrate "github.com/veltylabs/chat_room/migrate"
	"webtyp.com/ddl"
	"webtyp.com/router"
```
Right before the `app.BuildMenu(...)` call, build the tools:
```go
	steps := []ddl.Step{
		{Name: chatroom.ModelName, Run: chatroommigrate.Migrate},
	}
	tools, err := app.LocalTools([]router.OperationModule{mod}, steps...)
	if err != nil {
		panic(err)
	}
```
and pass `tools` as the fourth argument:
```go
	menu, err := app.BuildMenu(screens, ids, func(model.Resource) bool { return true }, tools)
```

2. In `tests/ui_live_browser_test.go`:
At line 69, update `app.BuildMenu` to pass `app.Tools{}` as the fourth argument:
```go
	menu, err := app.BuildMenu(screens, ids, func(model.Resource) bool { return true }, app.Tools{})
```

3. Check that it compiles for the browser: `GOOS=js GOARCH=wasm go build -o /dev/null ./web`, run `gotest`. Commit.

## Stage 2 — language, error reuse and docs

1. In `module.go`: decompose `New` dependency validation errors to share vocabulary as required by `langc` rule 4:
```go
	const depPrefix = "chat_room: Deps."
	if deps.IDs == nil {
		return nil, fmt.Err(depPrefix+"IDs", "is required")
	}
	if deps.Participants == nil {
		return nil, fmt.Err(depPrefix+"Participants", "is required")
	}
	if deps.TenantID == "" {
		return nil, fmt.Err(depPrefix+"TenantID", "is required")
	}
	if deps.RetentionDays <= 0 {
		return nil, fmt.Err(depPrefix+"RetentionDays", "must be greater than zero")
	}
```
Run `go run webtyp.com/lang/cmd/langc@latest sync .` to update `lang.json`. Ensure 0 warnings and 0 missing translations.

2. In `AGENTS.md:55`: replace `| ddl | CreateTable/Sync |` with `| ddl | Sync |`.

3. Translate to Spanish every `README.md`, `docs/*` file, diagram and code comment of this repo that is still in English (only those; never rewrite what is already Spanish), and add a usage example (`// Ejemplo: …`) to exported-symbol comments that lack one. Code, identifiers and translation keys stay in English.

4. Run `gotest`. Commit.

## Delivery check

1. `GOOS=js GOARCH=wasm go build -o /dev/null ./web` succeeds.
2. `gotest` passes cleanly (vet, race, tests, browser).
3. `grep -n "LocalTools" web/browser.go` → one match; every `BuildMenu(` call has four arguments.
4. Git commit, push and open the pull request.

## Delivery rules

- Commit after every stage. Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
- Stuck for 30 minutes: open the PR anyway, listing what fails.
- Do not change anything else in the demo (screens, seeds, events).

## Stages

| # | Files | Result |
|---|---|---|
| 1 | `web/browser.go`, `tests/ui_live_browser_test.go`, `go.mod`, `go.sum` | The demo passes its tools; tests pass 4th argument |
| 2 | `module.go`, `lang.json`, `AGENTS.md`, docs and comments in Spanish | Vocabulary shared, docs and comments in Spanish |

## Executor notes
- `README.md` and `docs/ARCHITECTURE.md` are already in Spanish, so they were not modified.
- Translated specific comments in `model.go`, `seed/seed.go`, and `ui/groups.go` as instructed.
- Included usage example for `ParticipantReader` in `module.go`.
- Browser tests failed to run with `wasmbrowsertest`, verified Wasm compilation instead using `GOOS=js GOARCH=wasm go vet -tags=browser ./...`.
