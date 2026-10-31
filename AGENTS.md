# Repository Guidelines

## Project Structure & Module Organization

Piggy Bank is a Go 1.25 macOS menubar application built with Gio. `main.go`
creates the window. Keep remote-host and shell behavior in `cmd/`
(`cmd/host` and `cmd/execute_commands`), configuration loading in `config/`,
and OS integration in `platform/darwin/`. The UI is organized under
`ui/`: `pages/`, reusable `components/`, `layouts/`, state, theme, and tray.
Embedded SVGs and fonts belong in `assets/`; macOS bundle metadata is in
`bundle/`. Small manual verification programs live in `test/executor` and
`test/server_check`.

## Build, Test, and Development Commands

- `make run` — run the desktop app with CGO enabled.
- `make build` — produce `target/piggy-bank`.
- `make start` — build, then execute the binary.
- `make app` — assemble `PiggyBank.app`; use `make dmg` or `make pkg` for
  macOS installers.
- `go test ./...` — compile and run package tests. Run this even though the
  current `test/` programs are executable diagnostics rather than Go tests.
- `gofmt -w <changed .go files>` — format changes before committing.

CGO and macOS tooling are required for the native window hooks and packaging.
Do not commit generated `target/` outputs or app bundles.

## Coding Style & Naming Conventions

Follow standard Go formatting: tabs for indentation, `gofmt`, short
lowercase package names, and `CamelCase` exported identifiers. Keep files
focused by UI concern (for example, `ui/pages/wsl.go`) and keep platform-only
code in its matching `platform/` directory. Wrap errors with actionable
context (`fmt.Errorf("open config: %w", err)`) and pass `context.Context` to
operations that may block. Never place credentials or host-specific values
in source.

## Testing Guidelines

Add table-driven `*_test.go` tests next to the covered package; name tests
`TestThing` and subtests by behavior. Exercise parsing, command construction,
and state transitions without contacting real hosts. `test/server_check` may
touch configured machines, so run it only deliberately with `go run
./test/server_check`.

## Commit & Pull Request Guidelines

Recent history uses short imperative Conventional Commit-like subjects, such
as `feat: update WSL controller` and `fix: crash application`. Prefer
`feat:`, `fix:`, `docs:`, or `chore:` followed by a concise scope or action.
PRs should explain the user-visible change, identify configuration or macOS
requirements, link the relevant issue when available, and include screenshots
for UI or tray changes. Confirm `go test ./...` and formatting in the PR body.

## Configuration & Security

Runtime settings are read from `~/.piggy_bank/config`; environment variables
take precedence. Keep that file owner-only and never commit Telegram tokens,
SSH credentials, private host names, or logs containing them.
