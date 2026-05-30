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
		OK:         true,
		Results:    []output.ResultItem{{Source: "telegram", Channel: "@test", MessageID: 1, Text: "hello"}},
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
