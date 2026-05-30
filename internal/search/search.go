package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	gotd "github.com/gotd/td/telegram"
	"github.com/heliofernandes404/tg-alerts/internal/output"
	"github.com/heliofernandes404/tg-alerts/internal/parser"
	"github.com/heliofernandes404/tg-alerts/internal/telegram"
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
// Per spec: if a filter is set and the field is empty (parser couldn't extract), no match.
func MatchesFilters(item FilterInput, req Request) bool {
	match := func(filter, field string) bool {
		if filter == "" {
			return true
		}
		if field == "" {
			return false
		}
		return strings.EqualFold(filter, field)
	}
	return match(req.Alertname, item.Alertname) &&
		match(req.Status, item.Status) &&
		match(req.Severity, item.Severity) &&
		match(req.Host, item.Host) &&
		match(req.Instance, item.Instance)
}

// Executor wraps gotd/td credentials for search operations.
type Executor struct {
	apiID   int
	apiHash string
	session gotd.SessionStorage
}

// NewExecutor builds an Executor from credentials and a session storage.
func NewExecutor(apiID int, apiHash string, session gotd.SessionStorage) *Executor {
	return &Executor{apiID: apiID, apiHash: apiHash, session: session}
}

// Execute runs the search and returns the full response plus any warnings.
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
			})
		}
		if err != nil {
			return err
		}

		results := make([]output.ResultItem, 0, len(messages))
		scanned := 0
		lastID := 0
		lowConfSeen := false

		for _, msg := range messages {
			if req.MaxScan > 0 && scanned >= req.MaxScan {
				warnings = appendUnique(warnings, "MAX_SCAN_REACHED")
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
			if req.Limit > 0 && len(results) >= req.Limit {
				break
			}

			if parsed.Confidence == "low" {
				lowConfSeen = true
			}
			results = append(results, output.ResultItem{
				Source:    "telegram",
				Channel:   ch.DisplayName,
				MessageID: msg.ID,
				Timestamp: msg.Timestamp.Format(time.RFC3339),
				Text:      msg.Text,
				Fields:    fieldsToOutput(parsed.Fields),
				Parse: output.ParseMeta{
					Confidence:      parsed.Confidence,
					MatchedPatterns: parsed.MatchedPatterns,
					MissingFields:   parsed.MissingFields,
				},
			})
		}

		if lowConfSeen {
			warnings = appendUnique(warnings, "PARSE_LOW_CONFIDENCE")
		}

		hasMore := len(messages) >= req.PageSize && lastID > 0
		var pagination output.Pagination
		if hasMore {
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
			OK:         true,
			Query:      buildQueryMeta(req, ch.DisplayName),
			Results:    results,
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
		resp = output.SearchResponse{
			OK: true,
			Results: []output.ResultItem{{
				Source:    "telegram",
				Channel:   ch.DisplayName,
				MessageID: msg.ID,
				Timestamp: msg.Timestamp.Format(time.RFC3339),
				Text:      msg.Text,
				Fields:    fieldsToOutput(parsed.Fields),
				Parse: output.ParseMeta{
					Confidence:      parsed.Confidence,
					MatchedPatterns: parsed.MatchedPatterns,
					MissingFields:   parsed.MissingFields,
				},
			}},
		}
		return nil
	})
	return resp, err
}

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

func buildQueryMeta(req Request, channelName string) output.QueryMeta {
	optStr := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}
	var since string
	if req.Since != nil {
		since = req.Since.Format(time.RFC3339)
	}
	var untilPtr *string
	if req.Until != nil {
		s := req.Until.Format(time.RFC3339)
		untilPtr = &s
	}
	return output.QueryMeta{
		Channel: channelName,
		Text:    req.Query,
		Since:   since,
		Until:   untilPtr,
		Filters: output.FiltersMeta{
			Host:      optStr(req.Host),
			Instance:  optStr(req.Instance),
			Alertname: optStr(req.Alertname),
			Severity:  optStr(req.Severity),
			Status:    optStr(req.Status),
		},
		Limit: req.Limit,
	}
}

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}
