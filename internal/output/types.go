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
