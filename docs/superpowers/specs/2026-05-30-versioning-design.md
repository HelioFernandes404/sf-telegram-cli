# Complete Versioning Design

**Date:** 2026-05-30
**Project:** tg-alerts CLI
**Status:** Approved

## Summary

Add complete versioning to the `tg-alerts` CLI: binary version info via ldflags, Makefile targets for build/release/install, goreleaser for multi-arch Linux builds and GitHub Releases, GitHub Actions triggered on git tags, and CLAUDE.md with release docs.

## 1. Binary Version Info

New file `internal/version/version.go` exposes version metadata with three package-level variables set to defaults:

```go
var (
    version   = "dev"
    commit    = "none"
    buildDate = "unknown"
)
```

At build time, goreleaser and `make build` override these via ldflags:

```
-X github.com/heliofernandes404/tg-alerts/internal/version.Version={{.Version}}
-X github.com/heliofernandes404/tg-alerts/internal/version.Commit={{.Commit}}
-X github.com/heliofernandes404/tg-alerts/internal/version.Date={{.Date}}
```

Output of `tg-alerts --version`:

```
{"version":"v1.2.3","commit":"abc1234","date":"2026-05-30T00:00:00Z"}
```

## 2. Makefile

Targets:

| Target | Description |
|--------|-------------|
| `build` | `go build` with ldflags, version from `git describe --tags` (fallback: `dev`) |
| `release VERSION=x.y.z` | Creates tag `vx.y.z` and pushes to origin |
| `install` | Downloads latest binary from GitHub Releases to `/usr/local/bin/tg-alerts` |
| `clean` | Removes local binary |

`make release` validates that `VERSION` is set before tagging. `make install` uses `curl` + GitHub API to resolve the latest release URL, detects arch via `uname -m`, and requires `sudo`.

## 3. goreleaser

File `.goreleaser.yaml`:

- **Builds:** `linux/amd64`, `linux/arm64`
- **ldflags:** injects `version`, `commit`, `buildDate`
- **Archives:** `tg-alerts_linux_amd64.tar.gz`, `tg-alerts_linux_arm64.tar.gz` — each containing the binary
- **Checksums:** `sha256sums.txt`
- **Changelog:** auto-generated from conventional commits, grouped by `feat`, `fix`, `docs`, `chore`
- **GitHub Release:** created automatically with archives + checksums + changelog

## 4. GitHub Actions

File `.github/workflows/release.yml`:

- **Trigger:** `push` to tags matching `v*`
- **Steps:**
  1. `actions/checkout@v4` with `fetch-depth: 0` (full history for changelog)
  2. `actions/setup-go@v5` using Go version from `go.mod`
  3. `goreleaser/goreleaser-action@v6` with `GITHUB_TOKEN` (no manual secret needed)

## 5. CLAUDE.md

New project-level `CLAUDE.md` documenting:

- Module path and Go version
- Project structure overview
- Conventional commit types (`feat`, `fix`, `docs`, `chore`)
- Release workflow: `make release VERSION=x.y.z`
- Install: `make install`

## Files Created / Modified

| File | Action |
|------|--------|
| `internal/version/version.go` | New — version metadata + ldflags vars |
| `cmd/root.go` | Modified — register JSON `--version` output |
| `Makefile` | New |
| `.goreleaser.yaml` | New |
| `.github/workflows/release.yml` | New |
| `CLAUDE.md` | New |
