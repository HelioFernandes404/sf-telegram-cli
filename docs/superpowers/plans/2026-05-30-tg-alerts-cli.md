# tg-alerts CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a read-only Go CLI (`tg-alerts`) that authenticates as a Telegram user via MTProto and queries alert channels, emitting stable JSON/NDJSON output for LLMs and automation agents.

**Architecture:** Seven packages with strict boundaries — `config` (YAML + env loading), `output` (JSON/NDJSON serialization), `parser` (tolerant alert field extraction), `auth` (session lifecycle), `telegram` (gotd/td MTProto wrapper), `search` (orchestration + local filtering + pagination), and `cmd` (Cobra command definitions). The `telegram` package exposes a `Client` interface so tests can mock it without network calls.

**Tech Stack:** Go 1.22+, `github.com/gotd/td` v0.x (MTProto), `github.com/spf13/cobra` (CLI), `gopkg.in/yaml.v3` (config), `github.com/adrg/xdg` (platform-aware paths), `golang.org/x/term` (2FA password prompt without echo)

---

## File Map

```
tg-alerts/
├── main.go
├── go.mod
├── internal/
│   ├── config/
│   │   ├── config.go          # Config struct, Load(), ConfigPath(), SessionPath()
│   │   └── config_test.go
│   ├── output/
│   │   ├── types.go           # All JSON output structs
│   │   ├── output.go          # WriteJSON, WriteError, WriteNDJSON
│   │   └── output_test.go
│   ├── parser/
│   │   ├── patterns.go        # compiled regex vars
│   │   ├── parser.go          # Parse(text string) Result
│   │   └── parser_test.go
│   ├── auth/
│   │   ├── session.go         # FileSession implementing telegram.SessionStorage
│   │   ├── auth.go            # Service: Login, Status, Logout
│   │   └── auth_test.go
│   ├── telegram/
│   │   ├── client.go          # NewClient() + Client interface
│   │   ├── channel.go         # ResolveChannel()
│   │   ├── history.go         # GetHistory(), Search(), GetMessage()
│   │   └── telegram_test.go   # mock-based tests
│   └── search/
│       ├── types.go           # Request struct
│       ├── pagination.go      # PageToken encode/decode/validate
│       ├── search.go          # Execute(), applyFilters()
│       └── search_test.go
└── cmd/
    ├── root.go                # rootCmd, persistent flags, Execute()
    ├── auth.go                # auth login / status / logout
    ├── search.go              # search command
    └── get.go                 # get command
```

---

## Task 1: Project scaffold

**Files:**
- Create: `main.go`
- Create: `go.mod`
- Create: `cmd/root.go`

- [ ] **Step 1: Initialise the module and install dependencies**

```bash
cd /home/helio/Obsidian/03-projetos/sftelegramcli
go mod init github.com/heliofernandes404/tg-alerts
go get github.com/spf13/cobra@latest
go get github.com/gotd/td@latest
go get gopkg.in/yaml.v3@latest
go get github.com/adrg/xdg@latest
go get golang.org/x/term@latest
go mod tidy
```

- [ ] **Step 2: Create `main.go`**

```go
package main

import "github.com/heliofernandes404/tg-alerts/cmd"

func main() {
	cmd.Execute()
}
```

- [ ] **Step 3: Create `cmd/root.go`**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tg-alerts",
	Short: "Query Telegram alert channels",
	Long:  "tg-alerts queries Telegram alert channels and returns JSON results for LLMs and automation agents.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Verify the project builds**

```bash
go build ./...
```

Expected: no errors, no output.

- [ ] **Step 5: Commit**

```bash
git add main.go go.mod go.sum cmd/root.go
git commit -m "feat: scaffold Go module and root cobra command"
```

---

## Task 2: config package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/config/config_test.go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/heliofernandes404/tg-alerts/internal/config"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	yaml := `
telegram:
  api_id: 12345
  api_hash_env: MY_HASH_ENV

defaults:
  channel: "@alerts"
  format: "json"
  limit: 25
`
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Telegram.APIID != 12345 {
		t.Errorf("api_id: got %d want 12345", cfg.Telegram.APIID)
	}
	if cfg.Telegram.APIHashEnv != "MY_HASH_ENV" {
		t.Errorf("api_hash_env: got %q", cfg.Telegram.APIHashEnv)
	}
	if cfg.Defaults.Channel != "@alerts" {
		t.Errorf("channel: got %q", cfg.Defaults.Channel)
	}
	if cfg.Defaults.Limit != 25 {
		t.Errorf("limit: got %d", cfg.Defaults.Limit)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := config.LoadFrom("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestAPIHashFromEnv(t *testing.T) {
	t.Setenv("TEST_API_HASH", "secrethash")
	dir := t.TempDir()
	yaml := "telegram:\n  api_id: 1\n  api_hash_env: TEST_API_HASH\n"
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	hash := cfg.APIHash()
	if hash != "secrethash" {
		t.Errorf("APIHash(): got %q want secrethash", hash)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/config/... -v
```

Expected: FAIL — package `config` not found.

- [ ] **Step 3: Implement `internal/config/config.go`**

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

type TelegramConfig struct {
	APIID      int    `yaml:"api_id"`
	APIHashEnv string `yaml:"api_hash_env"`
}

type Defaults struct {
	Channel string `yaml:"channel"`
	Format  string `yaml:"format"`
	Limit   int    `yaml:"limit"`
}

type Config struct {
	Telegram TelegramConfig `yaml:"telegram"`
	Defaults Defaults       `yaml:"defaults"`
}

// APIHash resolves the api_hash from the environment variable named in config.
func (c *Config) APIHash() string {
	return os.Getenv(c.Telegram.APIHashEnv)
}

// ConfigPath returns the platform-appropriate config file path.
func ConfigPath() string {
	return filepath.Join(xdg.ConfigHome, "tg-alerts", "config.yaml")
}

// SessionPath returns the platform-appropriate session file path.
func SessionPath() string {
	return filepath.Join(xdg.DataHome, "tg-alerts", "session.json")
}

// LoadFrom reads and parses the config file at the given path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return &cfg, nil
}

// Load reads from the default platform config path.
// Returns a zero-value Config (not an error) if the file does not yet exist.
func Load() (*Config, error) {
	path := ConfigPath()
	cfg, err := LoadFrom(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	return cfg, err
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/config/... -v
```

Expected: all three tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config package with YAML loading and platform-aware paths"
```

---

## Task 3: output types

**Files:**
- Create: `internal/output/types.go`

- [ ] **Step 1: Create `internal/output/types.go`**

```go
package output

// AlertFields holds structured fields extracted from an alert message.
type AlertFields struct {
	Alertname   string `json:"alertname,omitempty"`
	Status      string `json:"status,omitempty"`
	Severity    string `json:"severity,omitempty"`
	Host        string `json:"host,omitempty"`
	Instance    string `json:"instance,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
	Job         string `json:"job,omitempty"`
	Environment string `json:"environment,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
}

// ParseMeta describes the parser's confidence and pattern matches for a result.
type ParseMeta struct {
	Confidence      string   `json:"confidence"`
	MatchedPatterns []string `json:"matched_patterns"`
	MissingFields   []string `json:"missing_fields"`
}

// ResultItem is a single search result emitted to the caller.
type ResultItem struct {
	Source    string      `json:"source"`
	Channel   string      `json:"channel"`
	MessageID int         `json:"message_id"`
	Timestamp string      `json:"timestamp"` // RFC3339
	Text      string      `json:"text"`
	Fields    AlertFields `json:"fields"`
	Parse     ParseMeta   `json:"parse"`
}

// FiltersMeta reflects the structured filters used in the query.
type FiltersMeta struct {
	Host      *string `json:"host"`
	Instance  *string `json:"instance"`
	Alertname *string `json:"alertname"`
	Severity  *string `json:"severity"`
	Status    *string `json:"status"`
}

// QueryMeta reflects the effective query parameters back to the caller.
type QueryMeta struct {
	Channel string      `json:"channel"`
	Text    string      `json:"text,omitempty"`
	Since   string      `json:"since,omitempty"`
	Until   *string     `json:"until"`
	Filters FiltersMeta `json:"filters"`
	Limit   int         `json:"limit"`
}

// PaginationNext holds the opaque continuation token.
type PaginationNext struct {
	PageToken string `json:"page_token"`
}

// Pagination describes whether more results are available.
type Pagination struct {
	HasMore bool            `json:"has_more"`
	Next    *PaginationNext `json:"next"`
}

// SearchResponse is the top-level JSON success response for search and get.
type SearchResponse struct {
	OK         bool         `json:"ok"`
	Query      QueryMeta    `json:"query"`
	Results    []ResultItem `json:"results"`
	Warnings   []string     `json:"warnings,omitempty"`
	Pagination Pagination   `json:"pagination"`
}

// ErrorBody holds the structured error payload.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

// ErrorResponse is the top-level JSON error response.
type ErrorResponse struct {
	OK    bool      `json:"ok"`
	Error ErrorBody `json:"error"`
}

// AuthStatusAccount holds redacted account info for auth status.
type AuthStatusAccount struct {
	PhoneRedacted string `json:"phone_redacted"`
	Username      string `json:"username"`
}

// AuthStatusSession holds session file info for auth status.
type AuthStatusSession struct {
	Exists bool   `json:"exists"`
	Path   string `json:"path"`
}

// AuthStatusResponse is the output of auth status.
type AuthStatusResponse struct {
	OK            bool               `json:"ok"`
	Authenticated bool               `json:"authenticated"`
	Account       *AuthStatusAccount `json:"account,omitempty"`
	Session       AuthStatusSession  `json:"session"`
}

// NDJSONResult is a result item tagged for NDJSON mode.
type NDJSONResult struct {
	Type string `json:"type"` // "result"
	ResultItem
}

// NDJSONMeta is the terminal metadata object in NDJSON mode.
type NDJSONMeta struct {
	Type       string     `json:"type"` // "meta"
	OK         bool       `json:"ok"`
	Warnings   []string   `json:"warnings,omitempty"`
	Pagination Pagination `json:"pagination"`
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/output/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/output/types.go
git commit -m "feat: define all output types for JSON and NDJSON"
```

---

## Task 4: output writer

**Files:**
- Create: `internal/output/output.go`
- Create: `internal/output/output_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/output/output_test.go
package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/heliofernandes404/tg-alerts/internal/output"
)

func TestWriteErrorJSON(t *testing.T) {
	var buf bytes.Buffer
	output.WriteError(&buf, "AUTH_REQUIRED", "Session missing.", "Run tg-alerts auth login.")
	var resp output.ErrorResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if resp.OK {
		t.Error("ok should be false")
	}
	if resp.Error.Code != "AUTH_REQUIRED" {
		t.Errorf("code: got %q", resp.Error.Code)
	}
}

func TestWriteJSONSuccess(t *testing.T) {
	var buf bytes.Buffer
	resp := output.SearchResponse{
		OK:      true,
		Results: []output.ResultItem{{Source: "telegram", Channel: "@test", MessageID: 1, Text: "hello"}},
		Pagination: output.Pagination{HasMore: false},
	}
	if err := output.WriteJSON(&buf, resp); err != nil {
		t.Fatal(err)
	}
	var got output.SearchResponse
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if !got.OK {
		t.Error("ok should be true")
	}
	if len(got.Results) != 1 {
		t.Errorf("results len: got %d want 1", len(got.Results))
	}
}

func TestWriteNDJSON(t *testing.T) {
	var buf bytes.Buffer
	items := []output.ResultItem{
		{Source: "telegram", Channel: "@c", MessageID: 1, Text: "a"},
		{Source: "telegram", Channel: "@c", MessageID: 2, Text: "b"},
	}
	meta := output.NDJSONMeta{Type: "meta", OK: true, Pagination: output.Pagination{HasMore: false}}
	if err := output.WriteNDJSON(&buf, items, meta); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	// 2 results + 1 meta
	if len(lines) != 3 {
		t.Fatalf("expected 3 NDJSON lines, got %d:\n%s", len(lines), buf.String())
	}
	var lastLine map[string]any
	if err := json.Unmarshal([]byte(lines[2]), &lastLine); err != nil {
		t.Fatalf("last line not valid JSON: %v", err)
	}
	if lastLine["type"] != "meta" {
		t.Errorf("last line type: got %v want meta", lastLine["type"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/output/... -v
```

Expected: FAIL — `WriteError`, `WriteJSON`, `WriteNDJSON` not defined.

- [ ] **Step 3: Implement `internal/output/output.go`**

```go
package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// WriteJSON encodes v as indented JSON to w, followed by a newline.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// WriteError writes a structured JSON error response to w.
func WriteError(w io.Writer, code, message, hint string) {
	resp := ErrorResponse{
		OK: false,
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Hint:    hint,
		},
	}
	_ = WriteJSON(w, resp)
}

// WriteNDJSON writes each item as a tagged NDJSON result line, then meta.
func WriteNDJSON(w io.Writer, items []ResultItem, meta NDJSONMeta) error {
	enc := json.NewEncoder(w)
	for _, item := range items {
		row := NDJSONResult{Type: "result", ResultItem: item}
		if err := enc.Encode(row); err != nil {
			return fmt.Errorf("encoding ndjson result: %w", err)
		}
	}
	return enc.Encode(meta)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/output/... -v
```

Expected: all three tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/output/
git commit -m "feat: add output writer for JSON, NDJSON, and error responses"
```

---

## Task 5: parser patterns

**Files:**
- Create: `internal/parser/patterns.go`

- [ ] **Step 1: Create `internal/parser/patterns.go`**

```go
package parser

import "regexp"

// Each pattern captures one named group "val" for the field value.
// Patterns are ordered from most specific to least specific.
var (
	// "alertname: HighCPU", "Alert: HighCPU", "alert_name = HighCPU"
	reAlertname = regexp.MustCompile(`(?im)^[ \t]*(?:alertname|alert_name|alert)\s*[:=]\s*(?P<val>\S+)`)

	// "status: firing", "Status: RESOLVED", "🚨 FIRING" keyword lines
	reStatus = regexp.MustCompile(`(?im)(?:^[ \t]*status\s*[:=]\s*(?P<val>firing|resolved)|(?:^|\s)(?P<val>FIRING|RESOLVED)(?:\s|$))`)

	// "severity = critical", "Severity: critical", severity="critical"
	reSeverity = regexp.MustCompile(`(?im)^[ \t]*severity\s*[:=]["']?\s*(?P<val>critical|warning|info|page|none)["']?`)

	// "host: srv-01", "Host: srv-01"
	reHost = regexp.MustCompile(`(?im)^[ \t]*host\s*[:=]\s*(?P<val>\S+)`)

	// "instance: srv-01:9100"
	reInstance = regexp.MustCompile(`(?im)^[ \t]*instance\s*[:=]\s*(?P<val>\S+)`)

	// "summary: ..." or "summary = ..." captures rest of line
	reSummary = regexp.MustCompile(`(?im)^[ \t]*summary\s*[:=]\s*(?P<val>.+)$`)

	// "description: ..."
	reDescription = regexp.MustCompile(`(?im)^[ \t]*description\s*[:=]\s*(?P<val>.+)$`)

	// "job: ..."
	reJob = regexp.MustCompile(`(?im)^[ \t]*job\s*[:=]\s*(?P<val>\S+)`)

	// "environment: ..."
	reEnvironment = regexp.MustCompile(`(?im)^[ \t]*environment\s*[:=]\s*(?P<val>\S+)`)

	// "namespace: ..."
	reNamespace = regexp.MustCompile(`(?im)^[ \t]*namespace\s*[:=]\s*(?P<val>\S+)`)
)

// namedMatch returns the value of the named group "val" from the first match, or "".
func namedMatch(re *regexp.Regexp, text string) (string, string) {
	match := re.FindStringSubmatch(text)
	if match == nil {
		return "", ""
	}
	names := re.SubexpNames()
	for i, name := range names {
		if name == "val" && i < len(match) && match[i] != "" {
			return match[i], re.String()
		}
	}
	return "", ""
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/parser/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/parser/patterns.go
git commit -m "feat: add parser regex patterns for alert field extraction"
```

---

## Task 6: parser core

**Files:**
- Create: `internal/parser/parser.go`
- Create: `internal/parser/parser_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/parser/parser_test.go
package parser_test

import (
	"testing"

	"github.com/heliofernandes404/tg-alerts/internal/parser"
)

const tmplColon = `alertname: HighCPUUsage
status: firing
severity: critical
host: srv-01
instance: srv-01:9100
summary: CPU above threshold`

const tmplLabels = `Labels:
  alertname = DiskFull
  severity = warning
  instance = srv-02:9100

Annotations:
  summary = Disk usage above 85%`

const tmplEmoji = `🚨 FIRING
Alert: MemoryPressure
Host: srv-03
Severity: Critical`

func TestParseColonStyle(t *testing.T) {
	r := parser.Parse(tmplColon)
	if r.Fields.Alertname != "HighCPUUsage" {
		t.Errorf("alertname: got %q", r.Fields.Alertname)
	}
	if r.Fields.Status != "firing" {
		t.Errorf("status: got %q", r.Fields.Status)
	}
	if r.Fields.Severity != "critical" {
		t.Errorf("severity: got %q", r.Fields.Severity)
	}
	if r.Fields.Host != "srv-01" {
		t.Errorf("host: got %q", r.Fields.Host)
	}
	if r.Fields.Instance != "srv-01:9100" {
		t.Errorf("instance: got %q", r.Fields.Instance)
	}
	if r.Fields.Summary != "CPU above threshold" {
		t.Errorf("summary: got %q", r.Fields.Summary)
	}
	if r.Confidence != "high" {
		t.Errorf("confidence: got %q want high", r.Confidence)
	}
}

func TestParseLabelsBlock(t *testing.T) {
	r := parser.Parse(tmplLabels)
	if r.Fields.Alertname != "DiskFull" {
		t.Errorf("alertname: got %q", r.Fields.Alertname)
	}
	if r.Fields.Severity != "warning" {
		t.Errorf("severity: got %q", r.Fields.Severity)
	}
	if r.Fields.Instance != "srv-02:9100" {
		t.Errorf("instance: got %q", r.Fields.Instance)
	}
	if r.Fields.Summary != "Disk usage above 85%" {
		t.Errorf("summary: got %q", r.Fields.Summary)
	}
}

func TestParseEmojiStyle(t *testing.T) {
	r := parser.Parse(tmplEmoji)
	if r.Fields.Alertname != "MemoryPressure" {
		t.Errorf("alertname: got %q", r.Fields.Alertname)
	}
	if r.Fields.Status != "FIRING" && r.Fields.Status != "firing" {
		t.Errorf("status: got %q, want firing or FIRING", r.Fields.Status)
	}
	if r.Fields.Host != "srv-03" {
		t.Errorf("host: got %q", r.Fields.Host)
	}
	// severity: Critical → normalized to critical
	if r.Fields.Severity != "critical" {
		t.Errorf("severity: got %q want critical", r.Fields.Severity)
	}
}

func TestParseCaseNormalization(t *testing.T) {
	r := parser.Parse("severity: WARNING\nstatus: Resolved")
	if r.Fields.Severity != "warning" {
		t.Errorf("severity normalization: got %q want warning", r.Fields.Severity)
	}
	if r.Fields.Status != "resolved" {
		t.Errorf("status normalization: got %q want resolved", r.Fields.Status)
	}
}

func TestParseEmptyText(t *testing.T) {
	r := parser.Parse("")
	if r.Confidence != "low" {
		t.Errorf("empty text confidence: got %q want low", r.Confidence)
	}
	if r.Fields.Alertname != "" {
		t.Errorf("expected no alertname for empty text")
	}
}

func TestMissingFieldsTracked(t *testing.T) {
	r := parser.Parse("alertname: TestAlert\nseverity: critical")
	found := false
	for _, f := range r.MissingFields {
		if f == "status" {
			found = true
		}
	}
	if !found {
		t.Error("expected status in missing_fields")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/parser/... -v
```

Expected: FAIL — `parser.Parse` not defined.

- [ ] **Step 3: Implement `internal/parser/parser.go`**

```go
package parser

import "strings"

// Fields holds the structured alert fields extracted from a message.
type Fields struct {
	Alertname   string
	Status      string
	Severity    string
	Host        string
	Instance    string
	Summary     string
	Description string
	Job         string
	Environment string
	Namespace   string
}

// Result is the output of Parse.
type Result struct {
	Fields          Fields
	Confidence      string
	MatchedPatterns []string
	MissingFields   []string
}

var knownFields = []string{"alertname", "status", "severity", "host", "instance", "summary", "description", "job", "environment", "namespace"}

// Parse tolerantly extracts alert fields from raw Telegram message text.
func Parse(text string) Result {
	var r Result
	r.MatchedPatterns = []string{}
	r.MissingFields = []string{}

	extract := func(re interface {
		FindStringSubmatch(string) []string
		SubexpNames() []string
		String() string
	}, target *string, patternName string) {
		val, _ := namedMatch(re, text)
		if val != "" {
			*target = val
			r.MatchedPatterns = append(r.MatchedPatterns, patternName)
		}
	}

	extract(reAlertname, &r.Fields.Alertname, "alertname_label")
	extract(reHost, &r.Fields.Host, "host_candidate")
	extract(reInstance, &r.Fields.Instance, "instance_label")
	extract(reJob, &r.Fields.Job, "job_label")
	extract(reEnvironment, &r.Fields.Environment, "environment_label")
	extract(reNamespace, &r.Fields.Namespace, "namespace_label")
	extract(reSummary, &r.Fields.Summary, "summary_label")
	extract(reDescription, &r.Fields.Description, "description_label")

	// status: normalize to lowercase
	if raw, _ := namedMatch(reStatus, text); raw != "" {
		r.Fields.Status = strings.ToLower(raw)
		r.MatchedPatterns = append(r.MatchedPatterns, "status_keyword")
	}

	// severity: normalize to lowercase
	if raw, _ := namedMatch(reSeverity, text); raw != "" {
		r.Fields.Severity = strings.ToLower(raw)
		r.MatchedPatterns = append(r.MatchedPatterns, "severity_label")
	}

	r.MissingFields = missingFields(r.Fields)
	r.Confidence = confidence(r.Fields)
	return r
}

func missingFields(f Fields) []string {
	var missing []string
	check := map[string]string{
		"alertname": f.Alertname,
		"status":    f.Status,
		"severity":  f.Severity,
		"host":      f.Host,
		"instance":  f.Instance,
	}
	for _, k := range []string{"alertname", "status", "severity", "host", "instance"} {
		if check[k] == "" {
			missing = append(missing, k)
		}
	}
	if missing == nil {
		missing = []string{}
	}
	return missing
}

func confidence(f Fields) string {
	hasTarget := f.Host != "" || f.Instance != ""
	if f.Alertname != "" && f.Status != "" && hasTarget {
		return "high"
	}
	if f.Alertname != "" || f.Status != "" || f.Severity != "" {
		return "medium"
	}
	return "low"
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/parser/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/parser/
git commit -m "feat: add tolerant alert parser with confidence scoring"
```

---

## Task 7: auth session storage

**Files:**
- Create: `internal/auth/session.go`
- Create: `internal/auth/auth_test.go` (partial — session tests)

- [ ] **Step 1: Write failing tests for session storage**

```go
// internal/auth/auth_test.go
package auth_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/heliofernandes404/tg-alerts/internal/auth"
)

func TestFileSessionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	s := auth.NewFileSession(path)

	ctx := context.Background()

	// Load from non-existent file returns nil, no error
	data, err := s.LoadSession(ctx)
	if err != nil {
		t.Fatalf("LoadSession on missing file: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil for missing file, got %q", data)
	}

	// Store and reload
	want := []byte(`{"test":"data"}`)
	if err := s.StoreSession(ctx, want); err != nil {
		t.Fatalf("StoreSession: %v", err)
	}

	got, err := s.LoadSession(ctx)
	if err != nil {
		t.Fatalf("LoadSession after store: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestFileSessionPermissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission test not meaningful as root")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	s := auth.NewFileSession(path)
	_ = s.StoreSession(context.Background(), []byte("x"))

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions: got %o want 0600", perm)
	}
}

func TestSessionExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	s := auth.NewFileSession(path)

	if s.Exists() {
		t.Error("Exists() should be false before any store")
	}
	_ = s.StoreSession(context.Background(), []byte("x"))
	if !s.Exists() {
		t.Error("Exists() should be true after store")
	}
}

func TestSessionDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	s := auth.NewFileSession(path)
	_ = s.StoreSession(context.Background(), []byte("x"))

	if err := s.Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if s.Exists() {
		t.Error("Exists() should be false after Delete")
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/auth/... -v
```

Expected: FAIL — package `auth` not found.

- [ ] **Step 3: Implement `internal/auth/session.go`**

```go
package auth

import (
	"context"
	"os"
	"path/filepath"
	"sync"
)

// FileSession implements telegram.SessionStorage and provides helpers for the auth service.
type FileSession struct {
	path string
	mu   sync.Mutex
}

// NewFileSession creates a FileSession for the given path.
func NewFileSession(path string) *FileSession {
	return &FileSession{path: path}
}

// Path returns the session file path.
func (f *FileSession) Path() string { return f.path }

// LoadSession satisfies telegram.SessionStorage.
func (f *FileSession) LoadSession(_ context.Context) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := os.ReadFile(f.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// StoreSession satisfies telegram.SessionStorage. Writes with mode 0600.
func (f *FileSession) StoreSession(_ context.Context, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(f.path), 0700); err != nil {
		return err
	}
	return os.WriteFile(f.path, data, 0600)
}

// Exists reports whether the session file is present.
func (f *FileSession) Exists() bool {
	_, err := os.Stat(f.path)
	return err == nil
}

// Delete removes the session file.
func (f *FileSession) Delete() error {
	err := os.Remove(f.path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/auth/... -v
```

Expected: all four tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/auth/session.go internal/auth/auth_test.go
git commit -m "feat: add FileSession with restrictive permissions and lifecycle helpers"
```

---

## Task 8: auth service

**Files:**
- Modify: `internal/auth/auth_test.go` (add status/logout tests)
- Create: `internal/auth/auth.go`

- [ ] **Step 1: Add auth service tests to `internal/auth/auth_test.go`**

Append to the existing test file:

```go
func TestLogoutRemovesSession(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	os.WriteFile(sessionPath, []byte("data"), 0600)

	svc := auth.NewService(auth.ServiceConfig{SessionPath: sessionPath})
	if err := svc.Logout(); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := os.Stat(sessionPath); !os.IsNotExist(err) {
		t.Error("session file should be gone after Logout")
	}
}

func TestStatusNoSession(t *testing.T) {
	dir := t.TempDir()
	svc := auth.NewService(auth.ServiceConfig{
		SessionPath: filepath.Join(dir, "session.json"),
	})
	status := svc.LocalStatus()
	if status.Authenticated {
		t.Error("should not be authenticated with no session")
	}
	if status.Session.Exists {
		t.Error("session should not exist")
	}
}
```

- [ ] **Step 2: Run to verify new tests fail**

```bash
go test ./internal/auth/... -v -run "TestLogout|TestStatus"
```

Expected: FAIL — `auth.NewService`, `auth.ServiceConfig` not defined.

- [ ] **Step 3: Create `internal/auth/auth.go`**

```go
package auth

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"golang.org/x/term"
)

// ServiceConfig holds the dependencies for the auth service.
type ServiceConfig struct {
	SessionPath string
	APIID       int
	APIHash     string
}

// Service manages authentication lifecycle.
type Service struct {
	cfg     ServiceConfig
	session *FileSession
}

// NewService creates an auth Service.
func NewService(cfg ServiceConfig) *Service {
	return &Service{
		cfg:     cfg,
		session: NewFileSession(cfg.SessionPath),
	}
}

// LocalStatus returns auth status derived from the local session file only
// (no network call). Use this for the non-login parts of auth status.
func (s *Service) LocalStatus() LocalStatusResult {
	return LocalStatusResult{
		Authenticated: s.session.Exists(),
		Session: SessionInfo{
			Exists: s.session.Exists(),
			Path:   s.cfg.SessionPath,
		},
	}
}

// LocalStatusResult is returned by LocalStatus.
type LocalStatusResult struct {
	Authenticated bool
	Session       SessionInfo
}

// SessionInfo holds the session file status.
type SessionInfo struct {
	Exists bool
	Path   string
}

// Logout removes the local session file.
func (s *Service) Logout() error {
	return s.session.Delete()
}

// Login runs an interactive MTProto login flow, prompting on stderr.
// It writes the session file on success.
func (s *Service) Login(ctx context.Context) error {
	client := telegram.NewClient(s.cfg.APIID, s.cfg.APIHash, telegram.Options{
		SessionStorage: s.session,
	})

	return client.Run(ctx, func(ctx context.Context) error {
		status, err := client.Auth().Status(ctx)
		if err != nil {
			return fmt.Errorf("checking auth status: %w", err)
		}
		if status.Authorized {
			fmt.Fprintln(os.Stderr, "Already authenticated.")
			return nil
		}

		flow := auth.NewFlow(
			&consoleAuthenticator{},
			auth.SendCodeOptions{},
		)
		return client.Auth().IfNecessary(ctx, flow)
	})
}

// GetAccountInfo fetches the authenticated account's username and phone (redacted).
// Returns empty strings if the session is not authenticated.
func (s *Service) GetAccountInfo(ctx context.Context) (username, phoneRedacted string, err error) {
	client := telegram.NewClient(s.cfg.APIID, s.cfg.APIHash, telegram.Options{
		SessionStorage: s.session,
	})
	err = client.Run(ctx, func(ctx context.Context) error {
		self, err := client.Self(ctx)
		if err != nil {
			return err
		}
		username = self.Username
		phoneRedacted = redactPhone(self.Phone)
		return nil
	})
	return
}

// consoleAuthenticator prompts the operator on stderr for phone, code, and 2FA password.
type consoleAuthenticator struct {
	phone string
}

func (c *consoleAuthenticator) Phone(_ context.Context) (string, error) {
	if c.phone != "" {
		return c.phone, nil
	}
	fmt.Fprint(os.Stderr, "Phone number (+CCNUMBER): ")
	var phone string
	fmt.Fscan(os.Stdin, &phone)
	c.phone = strings.TrimSpace(phone)
	return c.phone, nil
}

func (c *consoleAuthenticator) Password(_ context.Context) (string, error) {
	fmt.Fprint(os.Stderr, "2FA password: ")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

func (c *consoleAuthenticator) Code(_ context.Context, _ interface{}) (string, error) {
	fmt.Fprint(os.Stderr, "Telegram code: ")
	var code string
	fmt.Fscan(os.Stdin, &code)
	return strings.TrimSpace(code), nil
}

func (c *consoleAuthenticator) AcceptTermsOfService(_ context.Context, _ interface{}) error {
	return nil
}

func (c *consoleAuthenticator) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("sign-up is not supported by tg-alerts")
}

var reDigits = regexp.MustCompile(`\d`)

// redactPhone masks digits except the last 4, e.g. "+55******1234".
func redactPhone(phone string) string {
	runes := []rune(phone)
	digitCount := 0
	for _, r := range runes {
		if r >= '0' && r <= '9' {
			digitCount++
		}
	}
	keep := 4
	masked := 0
	out := make([]rune, 0, len(runes))
	for _, r := range runes {
		if r >= '0' && r <= '9' {
			if digitCount-masked > keep {
				out = append(out, '*')
				masked++
			} else {
				out = append(out, r)
			}
		} else {
			out = append(out, r)
		}
	}
	_ = reDigits
	return string(out)
}
```

**Note:** `consoleAuthenticator.Code` and `AcceptTermsOfService` take `interface{}` parameters here because the exact gotd/td type signature depends on the installed version. After running `go get`, replace `interface{}` with the actual types from `github.com/gotd/td/telegram/auth` (likely `*tg.AuthSentCode` and `tg.HelpTermsOfService`). Run `go doc github.com/gotd/td/telegram/auth UserAuthenticator` to check the exact interface.

- [ ] **Step 4: Fix interface types after checking gotd/td docs**

```bash
go doc github.com/gotd/td/telegram/auth UserAuthenticator 2>/dev/null || go doc github.com/gotd/td/telegram/auth
```

Update `Code` and `AcceptTermsOfService` parameter types to match the actual `auth.UserAuthenticator` interface.

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/auth/... -v
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/auth/auth.go internal/auth/auth_test.go
git commit -m "feat: add auth service with login, logout, and local status"
```

---

## Task 9: telegram client and channel resolution

**Files:**
- Create: `internal/telegram/client.go`
- Create: `internal/telegram/channel.go`
- Create: `internal/telegram/telegram_test.go`

- [ ] **Step 1: Write tests**

```go
// internal/telegram/telegram_test.go
package telegram_test

import (
	"testing"

	"github.com/heliofernandes404/tg-alerts/internal/telegram"
)

// StripAt tests the channel name normalisation helper.
func TestStripAtPrefix(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"@canal-alertas", "canal-alertas"},
		{"canal-alertas", "canal-alertas"},
		{"@test", "test"},
	}
	for _, c := range cases {
		got := telegram.StripAt(c.in)
		if got != c.want {
			t.Errorf("StripAt(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/telegram/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Create `internal/telegram/client.go`**

```go
package telegram

import (
	"github.com/gotd/td/telegram"
)

// ClientOptions holds everything needed to build an MTProto client.
type ClientOptions struct {
	APIID          int
	APIHash        string
	SessionStorage telegram.SessionStorage
}

// NewClient returns a configured gotd/td Telegram client.
func NewClient(opts ClientOptions) *telegram.Client {
	return telegram.NewClient(opts.APIID, opts.APIHash, telegram.Options{
		SessionStorage: opts.SessionStorage,
	})
}
```

- [ ] **Step 4: Create `internal/telegram/channel.go`**

```go
package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/gotd/td/tg"
)

// StripAt removes a leading "@" from a channel username.
func StripAt(s string) string {
	return strings.TrimPrefix(s, "@")
}

// ResolvedChannel holds the resolved peer and access hash needed for API calls.
type ResolvedChannel struct {
	InputPeer    tg.InputPeerClass
	InputChannel *tg.InputChannel
}

// ResolveChannel resolves a channel username to its Telegram peer representation.
// username may include or omit the leading "@".
func ResolveChannel(ctx context.Context, api *tg.Client, username string) (*ResolvedChannel, error) {
	username = StripAt(username)
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		return nil, fmt.Errorf("resolving channel @%s: %w", username, err)
	}

	for _, chat := range resolved.Chats {
		switch c := chat.(type) {
		case *tg.Channel:
			return &ResolvedChannel{
				InputPeer: &tg.InputPeerChannel{
					ChannelID:  c.ID,
					AccessHash: c.AccessHash,
				},
				InputChannel: &tg.InputChannel{
					ChannelID:  c.ID,
					AccessHash: c.AccessHash,
				},
			}, nil
		}
	}
	return nil, fmt.Errorf("channel @%s not found in resolved result", username)
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/telegram/... -v
```

Expected: `TestStripAtPrefix` PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/telegram/
git commit -m "feat: add telegram client factory and channel resolution"
```

---

## Task 10: telegram history and search

**Files:**
- Create: `internal/telegram/history.go`

- [ ] **Step 1: Create `internal/telegram/history.go`**

```go
package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/gotd/td/tg"
)

// Message is a raw Telegram message returned from history or search calls.
type Message struct {
	ID        int
	Timestamp time.Time
	Text      string
}

// HistoryParams controls a messages.getHistory call.
type HistoryParams struct {
	Channel    *ResolvedChannel
	Limit      int
	OffsetID   int    // 0 = start from newest
	OffsetDate int    // 0 = no date restriction
	MinDate    int    // Unix timestamp lower bound (0 = none)
	MaxDate    int    // Unix timestamp upper bound (0 = none)
}

// SearchParams controls a messages.search call.
type SearchParams struct {
	Channel  *ResolvedChannel
	Query    string
	Limit    int
	OffsetID int
	MinDate  int
	MaxDate  int
}

// GetHistory fetches a page of channel history.
func GetHistory(ctx context.Context, api *tg.Client, p HistoryParams) ([]Message, error) {
	result, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:       p.Channel.InputPeer,
		OffsetID:   p.OffsetID,
		OffsetDate: p.OffsetDate,
		AddOffset:  0,
		Limit:      p.Limit,
		MaxID:      0,
		MinID:      0,
		Hash:       0,
	})
	if err != nil {
		return nil, fmt.Errorf("getHistory: %w", err)
	}
	return extractMessages(result), nil
}

// Search performs a text search within a channel.
func Search(ctx context.Context, api *tg.Client, p SearchParams) ([]Message, error) {
	result, err := api.MessagesSearch(ctx, &tg.MessagesSearchRequest{
		Peer:      p.Channel.InputPeer,
		Q:         p.Query,
		Filter:    &tg.InputMessagesFilterEmpty{},
		MinDate:   p.MinDate,
		MaxDate:   p.MaxDate,
		OffsetID:  p.OffsetID,
		AddOffset: 0,
		Limit:     p.Limit,
		MaxID:     0,
		MinID:     0,
		Hash:      0,
	})
	if err != nil {
		return nil, fmt.Errorf("messagesSearch: %w", err)
	}
	return extractMessages(result), nil
}

// GetMessage fetches a single message by ID from a channel.
func GetMessage(ctx context.Context, api *tg.Client, ch *ResolvedChannel, messageID int) (*Message, error) {
	result, err := api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
		Channel: ch.InputChannel,
		ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}},
	})
	if err != nil {
		return nil, fmt.Errorf("getMessages id=%d: %w", messageID, err)
	}
	msgs := extractMessages(result)
	if len(msgs) == 0 {
		return nil, fmt.Errorf("message %d not found", messageID)
	}
	return &msgs[0], nil
}

// extractMessages unwraps the union type returned by Telegram message API calls.
func extractMessages(v tg.MessagesMessagesClass) []Message {
	var raw []tg.MessageClass
	switch m := v.(type) {
	case *tg.MessagesMessages:
		raw = m.Messages
	case *tg.MessagesMessagesSlice:
		raw = m.Messages
	case *tg.MessagesChannelMessages:
		raw = m.Messages
	case *tg.MessagesMessagesNotModified:
		return nil
	}
	out := make([]Message, 0, len(raw))
	for _, item := range raw {
		if msg, ok := item.(*tg.Message); ok && msg.Message != "" {
			out = append(out, Message{
				ID:        msg.ID,
				Timestamp: time.Unix(int64(msg.Date), 0).UTC(),
				Text:      msg.Message,
			})
		}
	}
	return out
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/telegram/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/telegram/history.go
git commit -m "feat: add telegram history, search, and single-message fetch"
```

---

## Task 11: search pagination

**Files:**
- Create: `internal/search/types.go`
- Create: `internal/search/pagination.go`
- Create: `internal/search/search_test.go` (partial)

- [ ] **Step 1: Write pagination tests**

```go
// internal/search/search_test.go
package search_test

import (
	"testing"
	"time"

	"github.com/heliofernandes404/tg-alerts/internal/search"
)

func TestPageTokenRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tok := search.PageToken{
		Channel:   "@canal-alertas",
		Query:     "cpu",
		OffsetID:  500,
		MinDate:   now.Unix(),
		MaxDate:   now.Add(24 * time.Hour).Unix(),
		PageSize:  100,
		Severity:  "critical",
		Status:    "firing",
		Host:      "srv-01",
		Instance:  "",
		Alertname: "HighCPU",
	}
	encoded := tok.Encode()
	if encoded == "" {
		t.Fatal("Encode returned empty string")
	}
	decoded, err := search.DecodePageToken(encoded)
	if err != nil {
		t.Fatalf("DecodePageToken: %v", err)
	}
	if decoded.Channel != tok.Channel {
		t.Errorf("channel: got %q want %q", decoded.Channel, tok.Channel)
	}
	if decoded.OffsetID != tok.OffsetID {
		t.Errorf("offsetID: got %d want %d", decoded.OffsetID, tok.OffsetID)
	}
	if decoded.Severity != tok.Severity {
		t.Errorf("severity: got %q", decoded.Severity)
	}
}

func TestPageTokenInvalidInput(t *testing.T) {
	_, err := search.DecodePageToken("not-base64!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestPageTokenMatchesRequest(t *testing.T) {
	tok := search.PageToken{
		Channel:  "@c",
		Query:    "cpu",
		PageSize: 100,
	}
	req := search.Request{
		Channel:  "@c",
		Query:    "cpu",
		PageSize: 100,
	}
	if !tok.MatchesRequest(req) {
		t.Error("expected token to match request")
	}
	req.Channel = "@other"
	if tok.MatchesRequest(req) {
		t.Error("expected token to not match different channel")
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/search/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Create `internal/search/types.go`**

```go
package search

import "time"

// Request is the normalised set of search parameters.
type Request struct {
	Channel   string
	Query     string
	Since     *time.Time
	Until     *time.Time
	Host      string
	Instance  string
	Alertname string
	Severity  string
	Status    string
	Limit     int
	PageSize  int
	MaxScan   int
	Timeout   time.Duration
	Format    string
	PageToken string
}
```

- [ ] **Step 4: Create `internal/search/pagination.go`**

```go
package search

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// PageToken is the internal representation of a continuation token.
// It is serialised to opaque base64-JSON for callers.
type PageToken struct {
	Channel   string `json:"ch"`
	Query     string `json:"q,omitempty"`
	OffsetID  int    `json:"oid,omitempty"`
	MinDate   int64  `json:"min,omitempty"`
	MaxDate   int64  `json:"max,omitempty"`
	PageSize  int    `json:"ps"`
	Severity  string `json:"sev,omitempty"`
	Status    string `json:"sta,omitempty"`
	Host      string `json:"host,omitempty"`
	Instance  string `json:"inst,omitempty"`
	Alertname string `json:"an,omitempty"`
}

// Encode serialises the token to an opaque string safe for CLI flags.
func (t PageToken) Encode() string {
	data, _ := json.Marshal(t)
	return base64.RawURLEncoding.EncodeToString(data)
}

// DecodePageToken deserialises an opaque token string.
func DecodePageToken(s string) (PageToken, error) {
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return PageToken{}, fmt.Errorf("invalid page token encoding: %w", err)
	}
	var tok PageToken
	if err := json.Unmarshal(data, &tok); err != nil {
		return PageToken{}, fmt.Errorf("invalid page token content: %w", err)
	}
	return tok, nil
}

// MatchesRequest returns true if the token's invariant fields match the request.
// A mismatch means the token was issued for a different query and should be rejected.
func (t PageToken) MatchesRequest(r Request) bool {
	return t.Channel == r.Channel &&
		t.Query == r.Query &&
		t.PageSize == r.PageSize &&
		t.Severity == r.Severity &&
		t.Status == r.Status &&
		t.Host == r.Host &&
		t.Instance == r.Instance &&
		t.Alertname == r.Alertname
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/search/... -v -run "TestPageToken"
```

Expected: all three PageToken tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/search/types.go internal/search/pagination.go internal/search/search_test.go
git commit -m "feat: add search types and opaque page token encode/decode"
```

---

## Task 12: search execute

**Files:**
- Create: `internal/search/search.go`
- Modify: `internal/search/search_test.go` (add filter tests)

- [ ] **Step 1: Add filter tests to `internal/search/search_test.go`**

Append to the file:

```go
func TestApplyFiltersMatch(t *testing.T) {
	item := search.FilterInput{
		Alertname: "HighCPU",
		Status:    "firing",
		Severity:  "critical",
		Host:      "srv-01",
		Instance:  "srv-01:9100",
	}
	req := search.Request{
		Alertname: "highcpu", // case-insensitive
		Severity:  "critical",
		Status:    "firing",
		Host:      "srv-01",
	}
	if !search.MatchesFilters(item, req) {
		t.Error("expected match")
	}
}

func TestApplyFiltersMismatch(t *testing.T) {
	item := search.FilterInput{
		Alertname: "HighCPU",
		Severity:  "warning",
	}
	req := search.Request{Severity: "critical"}
	if search.MatchesFilters(item, req) {
		t.Error("expected no match — severity mismatch")
	}
}

func TestFilterFallback_MissingFieldNoMatch(t *testing.T) {
	// If parser couldn't extract a field and a filter is set, no match.
	item := search.FilterInput{
		Alertname: "", // parser didn't find it
	}
	req := search.Request{Alertname: "HighCPU"}
	if search.MatchesFilters(item, req) {
		t.Error("empty field with filter set should not match")
	}
}
```

- [ ] **Step 2: Run to verify new tests fail**

```bash
go test ./internal/search/... -v -run "TestApplyFilters|TestFilter"
```

Expected: FAIL — `search.FilterInput`, `search.MatchesFilters` not defined.

- [ ] **Step 3: Create `internal/search/search.go`**

```go
package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/heliofernandes404/tg-alerts/internal/output"
	"github.com/heliofernandes404/tg-alerts/internal/parser"
	"github.com/heliofernandes404/tg-alerts/internal/telegram"
	tgtd "github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

// FilterInput holds the parsed fields used for local filter matching.
type FilterInput struct {
	Alertname string
	Status    string
	Severity  string
	Host      string
	Instance  string
}

// MatchesFilters returns true if the item satisfies all active filters in req.
// Per spec: if a filter is set and the corresponding field is empty, no match.
func MatchesFilters(item FilterInput, req Request) bool {
	match := func(filter, field string) bool {
		if filter == "" {
			return true
		}
		if field == "" {
			return false // parser couldn't extract it → no match
		}
		return strings.EqualFold(filter, field)
	}
	return match(req.Alertname, item.Alertname) &&
		match(req.Status, item.Status) &&
		match(req.Severity, item.Severity) &&
		match(req.Host, item.Host) &&
		match(req.Instance, item.Instance)
}

// Result is returned by Execute.
type Result struct {
	Response output.SearchResponse
	Warnings []string
}

// Executor wraps a gotd/td client for search operations.
// Exposed as a struct so tests can substitute the client.
type Executor struct {
	apiID   int
	apiHash string
	session tgtd.SessionStorage
}

// NewExecutor builds an Executor from credentials and a session storage.
func NewExecutor(apiID int, apiHash string, session tgtd.SessionStorage) *Executor {
	return &Executor{apiID: apiID, apiHash: apiHash, session: session}
}

// Execute runs the search and returns the full response.
func (e *Executor) Execute(ctx context.Context, req Request, channelName string) (output.SearchResponse, []string, error) {
	client := telegram.NewClient(telegram.ClientOptions{
		APIID:          e.apiID,
		APIHash:        e.apiHash,
		SessionStorage: e.session,
	})

	var resp output.SearchResponse
	var warnings []string

	err := client.Run(ctx, func(ctx context.Context) error {
		api := client.API()

		ch, err := telegram.ResolveChannel(ctx, api, channelName)
		if err != nil {
			return fmt.Errorf("CHANNEL_NOT_FOUND: %w", err)
		}

		// Decode page token if provided
		var tok PageToken
		if req.PageToken != "" {
			tok, err = DecodePageToken(req.PageToken)
			if err != nil {
				return fmt.Errorf("QUERY_INVALID: %w", err)
			}
			if !tok.MatchesRequest(req) {
				return fmt.Errorf("QUERY_INVALID: page token does not match current query parameters")
			}
		}

		minDate, maxDate := timeBounds(req, tok)
		offsetID := tok.OffsetID

		var messages []telegram.Message
		if req.Query != "" {
			messages, err = telegram.Search(ctx, api, telegram.SearchParams{
				Channel:  ch,
				Query:    req.Query,
				Limit:    req.PageSize,
				OffsetID: offsetID,
				MinDate:  minDate,
				MaxDate:  maxDate,
			})
		} else {
			messages, err = telegram.GetHistory(ctx, api, telegram.HistoryParams{
				Channel:    ch,
				Limit:      req.PageSize,
				OffsetID:   offsetID,
				OffsetDate: maxDate,
				MinDate:    minDate,
				MaxDate:    maxDate,
			})
		}
		if err != nil {
			return err
		}

		results := make([]output.ResultItem, 0, len(messages))
		scanned := 0
		lastID := 0

		for _, msg := range messages {
			if scanned >= req.MaxScan {
				warnings = append(warnings, "MAX_SCAN_REACHED")
				break
			}
			scanned++
			lastID = msg.ID

			parsed := parser.Parse(msg.Text)
			fi := FilterInput{
				Alertname: parsed.Fields.Alertname,
				Status:    parsed.Fields.Status,
				Severity:  parsed.Fields.Severity,
				Host:      parsed.Fields.Host,
				Instance:  parsed.Fields.Instance,
			}
			if !MatchesFilters(fi, req) {
				continue
			}
			if len(results) >= req.Limit {
				break
			}

			lowConf := parsed.Confidence == "low"
			item := output.ResultItem{
				Source:    "telegram",
				Channel:   "@" + telegram.StripAt(channelName),
				MessageID: msg.ID,
				Timestamp: msg.Timestamp.Format(time.RFC3339),
				Text:      msg.Text,
				Fields:    fieldsToOutput(parsed.Fields),
				Parse: output.ParseMeta{
					Confidence:      parsed.Confidence,
					MatchedPatterns: parsed.MatchedPatterns,
					MissingFields:   parsed.MissingFields,
				},
			}
			results = append(results, item)
			if lowConf {
				warnings = append(warnings, "PARSE_LOW_CONFIDENCE")
			}
		}

		hasMore := len(messages) == req.PageSize && len(results) >= req.Limit
		var pagination output.Pagination
		if hasMore && lastID > 0 {
			nextTok := PageToken{
				Channel:   channelName,
				Query:     req.Query,
				OffsetID:  lastID,
				MinDate:   int64(minDate),
				MaxDate:   int64(maxDate),
				PageSize:  req.PageSize,
				Severity:  req.Severity,
				Status:    req.Status,
				Host:      req.Host,
				Instance:  req.Instance,
				Alertname: req.Alertname,
			}
			pagination = output.Pagination{
				HasMore: true,
				Next:    &output.PaginationNext{PageToken: nextTok.Encode()},
			}
		}

		resp = output.SearchResponse{
			OK:      true,
			Results: results,
			Pagination: pagination,
		}
		return nil
	})

	return resp, warnings, err
}

// ExecuteGetMessage fetches a single message by ID.
func (e *Executor) ExecuteGetMessage(ctx context.Context, channelName string, messageID int) (output.SearchResponse, error) {
	client := telegram.NewClient(telegram.ClientOptions{
		APIID:          e.apiID,
		APIHash:        e.apiHash,
		SessionStorage: e.session,
	})

	var resp output.SearchResponse
	err := client.Run(ctx, func(ctx context.Context) error {
		api := client.API()
		ch, err := telegram.ResolveChannel(ctx, api, channelName)
		if err != nil {
			return fmt.Errorf("CHANNEL_NOT_FOUND: %w", err)
		}
		msg, err := telegram.GetMessage(ctx, api, ch, messageID)
		if err != nil {
			return err
		}
		parsed := parser.Parse(msg.Text)
		item := output.ResultItem{
			Source:    "telegram",
			Channel:   "@" + telegram.StripAt(channelName),
			MessageID: msg.ID,
			Timestamp: msg.Timestamp.Format(time.RFC3339),
			Text:      msg.Text,
			Fields:    fieldsToOutput(parsed.Fields),
			Parse: output.ParseMeta{
				Confidence:      parsed.Confidence,
				MatchedPatterns: parsed.MatchedPatterns,
				MissingFields:   parsed.MissingFields,
			},
		}
		resp = output.SearchResponse{
			OK:      true,
			Results: []output.ResultItem{item},
		}
		return nil
	})
	return resp, err
}

// timeBounds converts Request time fields and a page token to Unix timestamps.
// Returns 0 for unset bounds.
func timeBounds(req Request, tok PageToken) (minDate, maxDate int) {
	if tok.MinDate != 0 {
		minDate = int(tok.MinDate)
	} else if req.Since != nil {
		minDate = int(req.Since.Unix())
	}
	if tok.MaxDate != 0 {
		maxDate = int(tok.MaxDate)
	} else if req.Until != nil {
		maxDate = int(req.Until.Unix())
	}
	return
}

// fieldsToOutput converts parser.Fields to output.AlertFields.
func fieldsToOutput(f parser.Fields) output.AlertFields {
	return output.AlertFields{
		Alertname:   f.Alertname,
		Status:      f.Status,
		Severity:    f.Severity,
		Host:        f.Host,
		Instance:    f.Instance,
		Summary:     f.Summary,
		Description: f.Description,
		Job:         f.Job,
		Environment: f.Environment,
		Namespace:   f.Namespace,
	}
}

// clientAPI is a helper to expose the tg.Client from a running gotd client.
// (gotd/td uses client.API() inside Run to get the raw tg.Client)
func clientAPI(c *tgtd.Client) *tg.Client {
	return c.API()
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go build ./internal/search/...
go test ./internal/search/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/search/
git commit -m "feat: add search executor with local filtering, max-scan, and pagination"
```

---

## Task 13: cmd auth subcommands

**Files:**
- Create: `cmd/auth.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Create `cmd/auth.go`**

```go
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/heliofernandes404/tg-alerts/internal/auth"
	"github.com/heliofernandes404/tg-alerts/internal/config"
	"github.com/heliofernandes404/tg-alerts/internal/output"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Telegram authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate your Telegram account locally",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			output.WriteError(os.Stdout, "CONFIG_INVALID", err.Error(), "")
			return nil
		}
		apiHash := cfg.APIHash()
		if apiHash == "" {
			output.WriteError(os.Stdout, "CONFIG_INVALID",
				fmt.Sprintf("environment variable %q is not set", cfg.Telegram.APIHashEnv),
				"Set the variable and try again.")
			return nil
		}
		svc := auth.NewService(auth.ServiceConfig{
			SessionPath: config.SessionPath(),
			APIID:       cfg.Telegram.APIID,
			APIHash:     apiHash,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := svc.Login(ctx); err != nil {
			output.WriteError(os.Stdout, "AUTH_REQUIRED", err.Error(), "")
			return nil
		}
		fmt.Fprintln(os.Stderr, "Login successful.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report local session status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.Load()
		sessionPath := config.SessionPath()
		svc := auth.NewService(auth.ServiceConfig{
			SessionPath: sessionPath,
			APIID:       cfg.Telegram.APIID,
			APIHash:     cfg.APIHash(),
		})
		local := svc.LocalStatus()
		resp := output.AuthStatusResponse{
			OK:            true,
			Authenticated: local.Authenticated,
			Session: output.AuthStatusSession{
				Exists: local.Session.Exists,
				Path:   local.Session.Path,
			},
		}
		// If authenticated, attempt to fetch account info via network.
		if local.Authenticated && cfg.Telegram.APIID != 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			username, phone, err := svc.GetAccountInfo(ctx)
			if err == nil {
				resp.Account = &output.AuthStatusAccount{
					PhoneRedacted: phone,
					Username:      username,
				}
			}
		}
		return output.WriteJSON(os.Stdout, resp)
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove the local Telegram session",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := auth.NewService(auth.ServiceConfig{
			SessionPath: config.SessionPath(),
		})
		if err := svc.Logout(); err != nil {
			output.WriteError(os.Stdout, "INTERNAL", err.Error(), "")
			return nil
		}
		fmt.Fprintln(os.Stderr, "Logged out. Session file removed.")
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd)
}
```

- [ ] **Step 2: Register authCmd in `cmd/root.go`**

In `cmd/root.go`, inside the `init()` function (add the function if it doesn't exist yet):

```go
func init() {
	rootCmd.AddCommand(authCmd)
}
```

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add cmd/auth.go cmd/root.go
git commit -m "feat: add auth login/status/logout subcommands"
```

---

## Task 14: cmd search command

**Files:**
- Create: `cmd/search.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Create `cmd/search.go`**

```go
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/heliofernandes404/tg-alerts/internal/auth"
	"github.com/heliofernandes404/tg-alerts/internal/config"
	"github.com/heliofernandes404/tg-alerts/internal/output"
	"github.com/heliofernandes404/tg-alerts/internal/search"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search a Telegram alert channel",
	RunE:  runSearch,
}

func init() {
	f := searchCmd.Flags()
	f.String("channel", "", "Telegram channel username (e.g. @canal-alertas)")
	f.String("query", "", "Text query (uses Telegram search when set)")
	f.String("since", "", "Lower time bound: duration like 24h or RFC3339 timestamp")
	f.String("until", "", "Upper time bound: duration like 24h or RFC3339 timestamp")
	f.String("host", "", "Filter by host field (exact, case-insensitive)")
	f.String("instance", "", "Filter by instance field")
	f.String("alertname", "", "Filter by alertname field")
	f.String("severity", "", "Filter by severity (critical, warning, info)")
	f.String("status", "", "Filter by status (firing, resolved)")
	f.Int("limit", 50, "Maximum results to return")
	f.Int("page-size", 100, "Telegram messages per request page")
	f.Int("max-scan", 1000, "Maximum messages to inspect locally")
	f.Duration("timeout", 30*time.Second, "Operation timeout")
	f.String("format", "json", "Output format: json or ndjson")
	f.String("page-token", "", "Continuation token from a previous response")
}

func runSearch(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		output.WriteError(os.Stdout, "CONFIG_INVALID", err.Error(), "")
		return nil
	}

	channel, _ := cmd.Flags().GetString("channel")
	if channel == "" {
		channel = cfg.Defaults.Channel
	}
	if channel == "" {
		output.WriteError(os.Stdout, "QUERY_INVALID", "--channel is required", "Set a default channel in config or pass --channel.")
		return nil
	}

	apiHash := cfg.APIHash()
	if apiHash == "" {
		output.WriteError(os.Stdout, "CONFIG_INVALID",
			fmt.Sprintf("env var %q not set", cfg.Telegram.APIHashEnv), "")
		return nil
	}

	session := auth.NewFileSession(config.SessionPath())
	if !session.Exists() {
		output.WriteError(os.Stdout, "AUTH_REQUIRED", "No local session found.", "Run tg-alerts auth login.")
		return nil
	}

	req := buildSearchRequest(cmd, cfg)

	timeout, _ := cmd.Flags().GetDuration("timeout")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ex := search.NewExecutor(cfg.Telegram.APIID, apiHash, session)
	resp, warnings, err := ex.Execute(ctx, req, channel)
	if err != nil {
		code, msg := classifyError(err)
		output.WriteError(os.Stdout, code, msg, "")
		return nil
	}
	resp.Warnings = deduplicateWarnings(warnings)

	format, _ := cmd.Flags().GetString("format")
	if format == "ndjson" {
		meta := output.NDJSONMeta{
			Type:       "meta",
			OK:         true,
			Warnings:   resp.Warnings,
			Pagination: resp.Pagination,
		}
		return output.WriteNDJSON(os.Stdout, resp.Results, meta)
	}
	return output.WriteJSON(os.Stdout, resp)
}

func buildSearchRequest(cmd *cobra.Command, cfg *config.Config) search.Request {
	limit, _ := cmd.Flags().GetInt("limit")
	if limit <= 0 {
		limit = cfg.Defaults.Limit
	}
	if limit <= 0 {
		limit = 50
	}
	pageSize, _ := cmd.Flags().GetInt("page-size")
	maxScan, _ := cmd.Flags().GetInt("max-scan")

	req := search.Request{
		Channel:   mustFlag(cmd, "channel"),
		Query:     mustFlag(cmd, "query"),
		Host:      mustFlag(cmd, "host"),
		Instance:  mustFlag(cmd, "instance"),
		Alertname: mustFlag(cmd, "alertname"),
		Severity:  mustFlag(cmd, "severity"),
		Status:    mustFlag(cmd, "status"),
		PageToken: mustFlag(cmd, "page-token"),
		Limit:     limit,
		PageSize:  pageSize,
		MaxScan:   maxScan,
	}

	if since := mustFlag(cmd, "since"); since != "" {
		t := parseBound(since, false)
		req.Since = &t
	}
	if until := mustFlag(cmd, "until"); until != "" {
		t := parseBound(until, true)
		req.Until = &t
	}
	return req
}

func mustFlag(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

// parseBound parses a duration (e.g. "24h") or RFC3339 string into a time.Time.
// isUpper=true means the duration is subtracted from now to get an upper bound.
func parseBound(s string, isUpper bool) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not parse time bound %q, ignoring\n", s)
		return time.Time{}
	}
	if isUpper {
		return time.Now().UTC()
	}
	return time.Now().UTC().Add(-d)
}

func classifyError(err error) (code, msg string) {
	s := err.Error()
	switch {
	case strings.HasPrefix(s, "CHANNEL_NOT_FOUND:"):
		return "CHANNEL_NOT_FOUND", strings.TrimPrefix(s, "CHANNEL_NOT_FOUND: ")
	case strings.HasPrefix(s, "QUERY_INVALID:"):
		return "QUERY_INVALID", strings.TrimPrefix(s, "QUERY_INVALID: ")
	case strings.Contains(s, "FLOOD_WAIT"):
		return "TELEGRAM_RATE_LIMITED", s
	case strings.Contains(s, "context deadline exceeded"):
		return "TIMEOUT", "Operation timed out."
	default:
		return "TELEGRAM_UNAVAILABLE", s
	}
}

func deduplicateWarnings(ws []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, w := range ws {
		if !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	return out
}
```

- [ ] **Step 2: Register searchCmd in `cmd/root.go`**

Add to the `init()` function in `cmd/root.go`:

```go
rootCmd.AddCommand(searchCmd)
```

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add cmd/search.go cmd/root.go
git commit -m "feat: add search command with all flags, local filters, and pagination"
```

---

## Task 15: cmd get command

**Files:**
- Create: `cmd/get.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Create `cmd/get.go`**

```go
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/heliofernandes404/tg-alerts/internal/auth"
	"github.com/heliofernandes404/tg-alerts/internal/config"
	"github.com/heliofernandes404/tg-alerts/internal/output"
	"github.com/heliofernandes404/tg-alerts/internal/search"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Fetch a single Telegram message by ID",
	RunE:  runGet,
}

func init() {
	f := getCmd.Flags()
	f.String("channel", "", "Telegram channel username (e.g. @canal-alertas)")
	f.Int("message-id", 0, "Telegram message ID to fetch")
	f.Duration("timeout", 30*time.Second, "Operation timeout")
}

func runGet(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		output.WriteError(os.Stdout, "CONFIG_INVALID", err.Error(), "")
		return nil
	}

	channel, _ := cmd.Flags().GetString("channel")
	if channel == "" {
		channel = cfg.Defaults.Channel
	}
	if channel == "" {
		output.WriteError(os.Stdout, "QUERY_INVALID", "--channel is required", "")
		return nil
	}

	messageID, _ := cmd.Flags().GetInt("message-id")
	if messageID <= 0 {
		output.WriteError(os.Stdout, "QUERY_INVALID", "--message-id must be a positive integer", "")
		return nil
	}

	apiHash := cfg.APIHash()
	if apiHash == "" {
		output.WriteError(os.Stdout, "CONFIG_INVALID",
			fmt.Sprintf("env var %q not set", cfg.Telegram.APIHashEnv), "")
		return nil
	}

	session := auth.NewFileSession(config.SessionPath())
	if !session.Exists() {
		output.WriteError(os.Stdout, "AUTH_REQUIRED", "No local session found.", "Run tg-alerts auth login.")
		return nil
	}

	timeout, _ := cmd.Flags().GetDuration("timeout")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ex := search.NewExecutor(cfg.Telegram.APIID, apiHash, session)
	resp, err := ex.ExecuteGetMessage(ctx, channel, messageID)
	if err != nil {
		code, msg := classifyError(err)
		output.WriteError(os.Stdout, code, msg, "")
		return nil
	}
	return output.WriteJSON(os.Stdout, resp)
}
```

- [ ] **Step 2: Register getCmd in `cmd/root.go`**

Add to the `init()` function in `cmd/root.go`:

```go
rootCmd.AddCommand(getCmd)
```

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add cmd/get.go cmd/root.go
git commit -m "feat: add get command for single message fetch"
```

---

## Task 16: final wiring, full build, and test run

**Files:**
- Modify: `cmd/root.go` (verify all commands registered)

- [ ] **Step 1: Confirm all commands are registered in `cmd/root.go`**

The final `init()` in `cmd/root.go` must read:

```go
func init() {
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(getCmd)
}
```

- [ ] **Step 2: Run full test suite**

```bash
go test ./... -v
```

Expected: all unit tests PASS. Tests in `internal/auth` and `internal/telegram` that require live Telegram credentials will be skipped or use mocks.

- [ ] **Step 3: Build the binary**

```bash
go build -o tg-alerts .
```

Expected: `tg-alerts` binary created with no errors.

- [ ] **Step 4: Smoke-test the CLI help output**

```bash
./tg-alerts --help
./tg-alerts auth --help
./tg-alerts search --help
./tg-alerts get --help
```

Expected: each command prints its usage with all documented flags.

- [ ] **Step 5: Verify stdout/stderr separation for errors**

```bash
./tg-alerts search --channel "@x" 2>/dev/null | python3 -m json.tool
```

Expected: valid JSON with `{"ok": false, "error": {...}}` on stdout even without a session.

- [ ] **Step 6: Add `tg-alerts` binary to `.gitignore`**

Create `.gitignore`:

```
tg-alerts
```

- [ ] **Step 7: Final commit**

```bash
git add cmd/root.go .gitignore
git commit -m "feat: wire all commands, verify full build and help output"
```

---

## Self-review against spec

### Spec coverage

| Requirement | Task |
|---|---|
| Go CLI, single binary | Task 1 |
| Local config + env-var api_hash | Task 2 |
| Platform-aware paths (xdg) | Task 2 |
| JSON output types + stable schema | Task 3 |
| WriteJSON / WriteError / WriteNDJSON | Task 4 |
| Parser: colon style, Labels block, emoji/keyword | Tasks 5–6 |
| Confidence levels (high/medium/low) | Task 6 |
| Case normalization severity/status | Task 6 |
| Filter fallback: missing field = no match | Task 12 |
| Session file with 0600 permissions | Task 7 |
| auth login (prompts to stderr, writes session) | Task 8 |
| auth status (redacted phone, session path) | Task 13 |
| auth logout (local session only) | Task 13 |
| search: --query uses messages.search | Task 12 |
| search: no --query uses getHistory | Task 12 |
| --limit vs --max-scan distinction | Task 12 |
| MAX_SCAN_REACHED warning | Task 12 |
| Pagination with opaque page token | Tasks 11, 12 |
| Page token validation against request | Task 11 |
| NDJSON output mode | Task 4, 14 |
| get command | Task 15 |
| Stable error codes (AUTH_REQUIRED, CHANNEL_NOT_FOUND…) | Tasks 13–15 |
| Secrets never on stdout/stderr | Tasks 7, 8, 13 |
| Read-only — no send/edit/delete | Entire plan |

### Placeholder scan

No TBD, TODO, or "implement later" markers found.

### Type consistency

- `parser.Fields` → `output.AlertFields` conversion happens in `search.go:fieldsToOutput()` — consistent across Tasks 6, 12.
- `search.FilterInput` populated from `parser.Fields` in `search.go:Execute()` — field names match Task 12 definition.
- `auth.NewFileSession()` used in Tasks 7, 13, 14, 15 — signature `NewFileSession(path string) *FileSession` consistent throughout.
- `search.NewExecutor()` used in Tasks 12, 14, 15 — signature `NewExecutor(apiID int, apiHash string, session tgtd.SessionStorage) *Executor` consistent.
- `classifyError()` defined in `cmd/search.go` and reused in `cmd/get.go` via the same `cmd` package — no duplication issue.
