# tg-alerts Development Instructions

## Project Purpose
- `tg-alerts` is a read-only Go CLI for querying Telegram alert channels, built for LLM/agent consumption.
- Automation contracts matter more than human UX: deterministic behavior, non-interactive paths where possible, JSON/NDJSON output, and stable command semantics.
- Human-friendly text may exist, but must not compromise machine-readable output or scriptability.

## Source Of Truth
- Module: `github.com/heliofernandes404/tg-alerts` from `go.mod`.
- Go: `1.26.3`.
- Entrypoint: `main.go` with Cobra subcommands under `cmd/`.
- Binary: `tg-alerts` from `make build`.
- Source of truth for development is `go.mod`, `Makefile`, Go source, tests, and release workflow.
- `CLAUDE.md` preserves useful structure/release facts, but usage docs and skills are not development source of truth.

## LLM-First CLI Contract
- Use JSON/NDJSON output for LLMs and automation agents.
- Keep stdout for command results only.
- Keep logs, diagnostics, and errors on stderr.
- Do not add prompts to query/search/get automation paths.
- Authentication may require Telegram interaction; keep non-auth commands deterministic and avoid hidden interactivity.
- Keep parser output and confidence scoring deterministic.
- `tg-alerts --version` must return JSON with `version`, `commit`, and `date` from `internal/version`.

## Architecture
- `cobra` subcommands live under `cmd/` and should only parse, validate, and delegate.
- `internal/auth` owns Telegram session management.
- `internal/config` owns YAML config and environment resolution.
- `internal/output` owns JSON/NDJSON writers.
- `internal/parser` owns tolerant alert parsing and confidence scoring.
- `internal/search` owns local filtering and pagination.
- `internal/telegram` owns gotd/td client integration, channel resolution, and history access.
- Keep Telegram network calls, session storage, config, and filesystem access behind testable seams.

## Development Workflow
- For behavior changes, start with a focused test or update an existing focused test first.
- Make the smallest code change that satisfies the behavior.
- Prefer mocks/fakes over real Telegram calls in tests.
- Keep parser/search tests deterministic and fixture-driven.
- Use conventional commits when committing after approval: `feat:`, `fix:`, `docs:`, or `chore:`.

## Commands
- Build: `make build` writes `./tg-alerts`.
- Clean: `make clean` removes `./tg-alerts`.
- Release: `make release VERSION_ARG=x.y.z` creates and pushes tag `vx.y.z`.
- Install: `make install` downloads the latest GitHub Release binary to `$(HOME)/.local/bin/tg-alerts` unless `INSTALL_DIR` is set.
- Test all packages: `go test ./...`.
- Focused test: `go test ./internal/<package> -run TestName` or `go test ./cmd -run TestName`.
- Format changed Go files: `gofmt -w <files>`.

## Validation
- Run focused tests first for behavior changes.
- Run `go test ./...` before handing off shared behavior changes.
- Run `go vet ./...` when touching logic, IO, or interfaces.
- Run `gofmt -w <files>` for Go edits.
- Safe/read-only smoke tests only, such as `./tg-alerts --version` after `make build`.
- Do not run commands that authenticate, create sessions, or contact Telegram unless the user approves.

## Safety
- The CLI is read-only for Telegram alert querying, but auth/session files and API credentials are sensitive.
- Do not commit Telegram sessions, config with secrets, API credentials, caches, local state, downloaded data, or generated binaries.
- Ask before authenticating, contacting Telegram, installing binaries, installing dependencies, committing, tagging, pushing, or releasing.
- Do not print session material or Telegram credentials.

## Release Workflow
- Never release without explicit user approval.
- Version data is injected with `git describe --tags --always --dirty` and Go `ldflags` into `internal/version`.
- Keep the version JSON contract stable: `tg-alerts --version` returns `version`, `commit`, and `date`.
- Approved release command: `make release VERSION_ARG=x.y.z`.
- The release target creates tag `vx.y.z` and pushes it to origin; GitHub Actions/Goreleaser builds linux/amd64 and linux/arm64 binaries and publishes a GitHub Release with changelog.
