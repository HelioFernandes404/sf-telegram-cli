# Complete Versioning Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full versioning to tg-alerts CLI: binary version info via ldflags, Makefile, goreleaser multi-arch builds, GitHub Actions release pipeline, and CLAUDE.md.

**Architecture:** `internal/version` holds ldflags-injected variables and `cmd/root.go` exposes them through Cobra's `--version` flag. A `Makefile` drives local dev and tag-based releases. `.goreleaser.yaml` defines the build matrix and GitHub Release publishing. GitHub Actions triggers goreleaser on every `v*` tag push.

**Tech Stack:** Go 1.26.3, cobra, goreleaser v2, GitHub Actions

---

## File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/version/version.go` | Create | ldflags vars |
| `cmd/version_test.go` | Create | verify `--version` JSON output |
| `Makefile` | Create | build/release/install/clean targets |
| `.goreleaser.yaml` | Create | linux/amd64+arm64 builds + GitHub Release |
| `.github/workflows/release.yml` | Create | CI: trigger goreleaser on `v*` tags |
| `CLAUDE.md` | Create | project docs for future sessions |

---

### Task 1: Binary `--version` output

**Files:**
- Create: `cmd/version.go`
- Create: `cmd/version_test.go`

- [ ] **Step 1: Write the failing test**

```go
// cmd/version_test.go
package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "version") || !strings.Contains(got, "commit") || !strings.Contains(got, "date") {
		t.Errorf("unexpected output: %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/helio/Obsidian/03-projetos/sftelegramcli
go test ./cmd/ -run TestVersionCommand -v
```

Expected: FAIL — `version` command not found or output doesn't match.

- [ ] **Step 3: Create `cmd/version.go`**

```go
package cmd

import "encoding/json"

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func init() {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(versionJSON())
}

func versionJSON() string { /* returns JSON for --version */ }
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./cmd/ -run TestVersionCommand -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/version.go cmd/version_test.go
git commit -m "feat: add json version flag with ldflags injection"
```

---

### Task 2: Makefile

**Files:**
- Create: `Makefile`

- [ ] **Step 1: Create Makefile**

```makefile
BINARY  := tg-alerts
MODULE  := github.com/heliofernandes404/tg-alerts

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +%Y-%m-%d)

LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: build clean release install

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

clean:
	rm -f $(BINARY)

release:
ifndef VERSION_ARG
	$(error VERSION_ARG is required. Usage: make release VERSION_ARG=1.0.0)
endif
	git tag v$(VERSION_ARG)
	git push origin v$(VERSION_ARG)

install:
	@ARCH=$$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/'); \
	URL=$$(curl -sf https://api.github.com/repos/heliofernandes404/tg-alerts/releases/latest \
	  | grep "browser_download_url" \
	  | grep "linux_$${ARCH}.tar.gz" \
	  | cut -d '"' -f 4); \
	if [ -z "$$URL" ]; then echo "Error: no release found for linux/$$ARCH" >&2; exit 1; fi; \
	echo "Downloading $$URL ..."; \
	curl -fL "$$URL" | sudo tar -xz -C /usr/local/bin $(BINARY); \
	echo "Installed $(BINARY) to /usr/local/bin"
```

> Note: `release` target uses `VERSION_ARG` (not `VERSION`) to avoid shadowing the `VERSION` variable computed from git describe.

- [ ] **Step 2: Verify build target works**

```bash
make build
./tg-alerts --version
```

Expected output is JSON with `version`, `commit`, and `date` keys populated.

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "build: add Makefile with build/release/install/clean targets"
```

---

### Task 3: goreleaser config

**Files:**
- Create: `.goreleaser.yaml`

- [ ] **Step 1: Create `.goreleaser.yaml`**

```yaml
version: 2

project_name: tg-alerts

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - "-s -w -X github.com/heliofernandes404/tg-alerts/internal/version.Version={{.Version}} -X github.com/heliofernandes404/tg-alerts/internal/version.Commit={{.ShortCommit}} -X github.com/heliofernandes404/tg-alerts/internal/version.Date={{.Date}}"

archives:
  - formats:
      - tar.gz
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
    files: []

checksum:
  name_template: sha256sums.txt
  algorithm: sha256

changelog:
  sort: asc
  filters:
    exclude:
      - "^Merge"
  groups:
    - title: Features
      regexp: "^feat"
      order: 0
    - title: Bug Fixes
      regexp: "^fix"
      order: 1
    - title: Documentation
      regexp: "^docs"
      order: 2
    - title: Other
      order: 999
```

- [ ] **Step 2: Validate goreleaser config (dry run)**

Install goreleaser if not present:
```bash
go install github.com/goreleaser/goreleaser/v2@latest
```

Then validate:
```bash
goreleaser check
```

Expected: `config is valid` (or similar success message, no errors).

- [ ] **Step 3: Commit**

```bash
git add .goreleaser.yaml
git commit -m "build: add goreleaser config for linux amd64/arm64 releases"
```

---

### Task 4: GitHub Actions release workflow

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Create directory and workflow file**

```bash
mkdir -p .github/workflows
```

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: add GitHub Actions release workflow triggered on v* tags"
```

---

### Task 5: CLAUDE.md

**Files:**
- Create: `CLAUDE.md`

- [ ] **Step 1: Create `CLAUDE.md`**

```markdown
# tg-alerts

Read-only Go CLI for querying Telegram alert channels. Returns JSON for LLMs and automation agents.

## Module

`github.com/heliofernandes404/tg-alerts` — Go 1.26.3

## Project Structure

```
cmd/          cobra subcommands (auth, search, get)
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
```

- [ ] **Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add CLAUDE.md with project structure and release workflow"
```

---

### Task 6: Smoke test end-to-end

- [ ] **Step 1: Run all tests**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 2: Build and check version output**

```bash
make build
./tg-alerts --version
```

Expected: JSON with `version`, `commit`, and `date`.

- [ ] **Step 3: List all subcommands**

```bash
./tg-alerts --help
```

Expected: `version` appears in available commands list.
