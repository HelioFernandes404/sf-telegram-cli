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
