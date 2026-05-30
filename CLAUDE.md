# tg-alerts

Read-only Go CLI for querying Telegram alert channels. Returns JSON for LLMs and automation agents.

## Module

`github.com/heliofernandes404/tg-alerts` — Go 1.26.3

## Project Structure

```
cmd/          cobra subcommands (auth, search, get, version)
internal/
  auth/       Telegram session management
  config/     YAML config + env var resolution
  output/     JSON/NDJSON writer
  parser/     tolerant alert parser with confidence scoring
  search/     local filtering, pagination
  telegram/   gotd/td client, channel resolution, history
main.go       entrypoint
```

## Conventional Commits

Always use:
- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation only
- `chore:` tooling, deps, config

These drive the auto-generated CHANGELOG in releases.

## Release Workflow

1. Merge all changes to `main`
2. Run: `make release VERSION_ARG=x.y.z`
   - Creates tag `vx.y.z` and pushes to origin
3. GitHub Actions detects the tag and runs goreleaser
4. Goreleaser builds linux/amd64 + linux/arm64 binaries and publishes a GitHub Release with changelog

## Install Latest Binary

```bash
make install
```

Downloads the latest GitHub Release binary for the current arch to `/usr/local/bin/tg-alerts`. Requires `curl` and `sudo`.

## Local Build

```bash
make build        # builds ./tg-alerts with version from git describe
make clean        # removes ./tg-alerts
```
