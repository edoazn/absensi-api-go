---
name: go
description: Use when writing, reviewing, or debugging Go code in this project (absensi-go). Covers project layout, idiomatic Go conventions, error handling, testing, and build/verify commands.
---

# Go Skill (absensi-go)

Module: `github.com/edoazn/absensi-go` (Go 1.25).

## Project layout
- `cmd/` – application entry points
- `internal/` – private application code
- `config/` – configuration loading (`.env`, see `.env.example`)
- `web/` – web assets/handlers
- `bin/` – build output (do not commit)

## Conventions
- Run `gofmt`/`go vet` before finishing; keep code idiomatic and simple.
- Handle every error; wrap with context: `fmt.Errorf("doing x: %w", err)`.
- Pass `context.Context` as first parameter for I/O and DB calls.
- Keep interfaces small and define them where they are consumed.
- No global mutable state; inject dependencies via constructors.
- Never hardcode secrets; read from env/config. Don't commit `.env`.
- Package names: short, lowercase, no underscores. Exported identifiers need doc comments.

## Testing
- Table-driven tests in `*_test.go` next to the code.
- Use `t.Helper()`, `t.Run`, and `t.Parallel()` where safe.

## Verify
```
go build ./...
go vet ./...
go test ./...
```
