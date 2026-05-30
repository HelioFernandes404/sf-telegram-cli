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

	rootCmd.AddCommand(searchCmd)
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
		output.WriteError(os.Stdout, "QUERY_INVALID", "--channel is required",
			"Set a default channel in config or pass --channel.")
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
		output.WriteError(os.Stdout, "AUTH_REQUIRED", "No local session found.",
			"Run tg-alerts auth login.")
		return nil
	}

	req := buildSearchRequest(cmd, cfg, channel)

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
	resp.Warnings = warnings

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

func buildSearchRequest(cmd *cobra.Command, cfg *config.Config, channel string) search.Request {
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
		Channel:   channel,
		Query:     flagStr(cmd, "query"),
		Host:      flagStr(cmd, "host"),
		Instance:  flagStr(cmd, "instance"),
		Alertname: flagStr(cmd, "alertname"),
		Severity:  flagStr(cmd, "severity"),
		Status:    flagStr(cmd, "status"),
		PageToken: flagStr(cmd, "page-token"),
		Limit:     limit,
		PageSize:  pageSize,
		MaxScan:   maxScan,
	}

	if since := flagStr(cmd, "since"); since != "" {
		t := parseBound(since, false)
		req.Since = &t
	}
	if until := flagStr(cmd, "until"); until != "" {
		t := parseBound(until, true)
		req.Until = &t
	}
	return req
}

func flagStr(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

// parseBound parses a duration (e.g. "24h") or RFC3339 string to a time.Time.
// isUpper=true: the bound is "now" (duration is irrelevant for upper bound without explicit value).
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
