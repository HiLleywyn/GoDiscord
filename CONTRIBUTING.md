# Contributing to GoDiscord

Thank you for your interest in GoDiscord! This document describes the process for proposing changes and the standards that apply.

---

## Ground Rules

- **Zero external dependencies.** GoDiscord uses only the Go standard library. PRs that add third-party imports will not be merged.
- **Go 1.21+.** The minimum required version is Go 1.21. Do not use language or library features introduced after 1.21 without bumping the requirement in `go.mod` and documenting the change.
- **Thread safety.** All exported methods must be safe for concurrent use. Use `sync.Mutex` / `sync.RWMutex` or `sync/atomic` as appropriate. Document any exceptions explicitly.

---

## Getting Started

```bash
# Fork and clone
git clone https://github.com/YOUR_USERNAME/GoDiscord.git
cd GoDiscord

# Verify the build — no setup required
GOWORK=off go build .

# Run vet and staticcheck
go vet ./...
```

There is no separate test suite at this time; the build and vet checks are the baseline.

---

## Development Workflow

1. **Create a branch** from `main` (or the current development branch):
   ```bash
   git checkout -b feat/my-feature
   ```

2. **Make your changes.** Follow the style and patterns already present in the codebase.

3. **Build and vet:**
   ```bash
   GOWORK=off go build .
   go vet ./...
   ```

4. **Open a pull request** against `main` with a clear description of what changed and why.

---

## Coding Standards

### Style

- Follow standard `gofmt` formatting. Run `gofmt -w .` before committing.
- Follow [Effective Go](https://go.dev/doc/effective_go) conventions.
- Keep exported symbols documented with `// Symbol` godoc comments.
- Comment non-obvious logic with `// why`, not `// what`.

### Error handling

- Use the `*APIError` type for REST failures — never `fmt.Errorf("discord api: ...")`.
- Return errors rather than panicking, except in `New()` for programmer-error preconditions (empty token).

### Concurrency

- Protect all shared state accessed from both the gateway goroutine and handler goroutines.
- When adding a new field to `gateway` that is read/written from multiple goroutines, protect it with `sessionMu` (for session state) or `sync/atomic` (for counters).
- When adding a new event handler type to `eventDispatcher`, wrap it in `safeGo()` in the dispatch switch.

### Adding a new event type

1. Add a handler type to `events.go`:
   ```go
   // XyzHandler is called when ...
   XyzHandler func(*Bot, *XyzEvent)
   ```
2. Add the field and `addXyz()` method to `eventDispatcher`.
3. Add the event type struct to `types.go`.
4. Add the dispatch case in `eventDispatcher.dispatch()` — use `safeGo`.
5. Add the `OnXyz()` registration method to `bot.go`.
6. Document the handler type in the `README.md` events table.

### Adding a new REST method

1. Add the method to `rest.go` in the appropriate section.
2. Return `*APIError` (via `r.do()`) — do not wrap errors with `fmt.Errorf`.
3. Add a godoc comment that includes the Discord endpoint and any Discord-imposed limits.
4. If the method requires a new struct, add it to `types.go`.

---

## Commit Messages

Use the conventional commit format:

```
feat: add FollowupMessage REST methods
fix: guard sessionID/resumeURL with sessionMu
docs: document Permission.Has() edge cases
refactor: extract backoffDelay into testable function
```

Keep the subject line under 72 characters. Use the body to explain the *why* when the commit is non-obvious.

---

## Pull Request Checklist

Before marking your PR ready for review:

- [ ] `GOWORK=off go build .` passes with no errors
- [ ] `go vet ./...` passes with no warnings
- [ ] All exported symbols have godoc comments
- [ ] Concurrency is correct (no data races under `-race`)
- [ ] `CHANGELOG.md` has an entry under `## [Unreleased]`
- [ ] No external imports have been added

---

## Reporting Issues

Open a GitHub Issue with:
- GoDiscord version / commit SHA
- Go version (`go version`)
- Minimal reproducer code or steps to reproduce
- Expected vs actual behaviour
