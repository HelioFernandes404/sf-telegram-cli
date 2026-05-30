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
	if r.Fields.Status != "firing" {
		t.Errorf("status: got %q, want firing", r.Fields.Status)
	}
	if r.Fields.Host != "srv-03" {
		t.Errorf("host: got %q", r.Fields.Host)
	}
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
