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
	item := search.FilterInput{
		Alertname: "", // parser didn't find it
	}
	req := search.Request{Alertname: "HighCPU"}
	if search.MatchesFilters(item, req) {
		t.Error("empty field with filter set should not match")
	}
}
