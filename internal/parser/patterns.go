package parser

import "regexp"

// Each pattern captures one named group "val" for the field value.
// Patterns allow an arbitrary non-letter prefix so they match both plain labels
// ("alertname: X") and bullet-point formats ("• alertname: X").
var (
	// "alertname: HighCPU", "• alertname: AP_Last_Contacted", "Alert: HighCPU"
	reAlertname = regexp.MustCompile(`(?im)^[^a-zA-Z]*(?:alertname|alert_name|alert)\s*[:=]\s*(?P<val>\S+)`)

	// "status: firing", "• status: firing"
	reStatusLabel = regexp.MustCompile(`(?im)^[^a-zA-Z]*status\s*[:=]\s*(?P<val>\S+)`)

	// standalone keyword on its own line, optionally after emoji/punctuation: "🚨 FIRING"
	reStatusKeyword = regexp.MustCompile(`(?m)^[^a-zA-Z0-9]*(?P<val>FIRING|RESOLVED)\s*$`)

	// header lines from systemframe template:
	//   "✅ Alerts Resolved ✅"  → resolved
	//   "🔥 Critical Alerts 🔥" → firing
	//   "⚠️ Warning Alerts ⚠️"  → firing
	reStatusResolved = regexp.MustCompile(`(?im)Alerts?\s+Resolved`)
	reStatusFiring   = regexp.MustCompile(`(?im)(?:Critical|Warning|Firing)\s+Alerts?|🔥`)

	// "severity = critical", "• severity: warning", severity="critical"
	reSeverity = regexp.MustCompile(`(?im)^[^a-zA-Z]*severity\s*[:=]["']?\s*(?P<val>critical|warning|info|page|none)["']?`)

	// "host: srv-01", "• host: F020-Tijuca-RJ"
	reHost = regexp.MustCompile(`(?im)^[^a-zA-Z]*host\s*[:=]\s*(?P<val>\S+)`)

	// "instance: srv-01:9100", "• instance: systemframe-prod"
	reInstance = regexp.MustCompile(`(?im)^[^a-zA-Z]*instance\s*[:=]\s*(?P<val>\S+)`)

	// summary / description capture rest of line
	reSummary     = regexp.MustCompile(`(?im)^[^a-zA-Z]*summary\s*[:=]\s*(?P<val>.+)$`)
	reDescription = regexp.MustCompile(`(?im)^[^a-zA-Z]*description\s*[:=]\s*(?P<val>.+)$`)

	reJob         = regexp.MustCompile(`(?im)^[^a-zA-Z]*job\s*[:=]\s*(?P<val>\S+)`)
	reEnvironment = regexp.MustCompile(`(?im)^[^a-zA-Z]*environment\s*[:=]\s*(?P<val>\S+)`)
	reNamespace   = regexp.MustCompile(`(?im)^[^a-zA-Z]*namespace\s*[:=]\s*(?P<val>\S+)`)
)

// namedMatch returns the value of the named group "val" from the first match, or "".
func namedMatch(re *regexp.Regexp, text string) string {
	match := re.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	for i, name := range re.SubexpNames() {
		if name == "val" && i < len(match) && match[i] != "" {
			return match[i]
		}
	}
	return ""
}
