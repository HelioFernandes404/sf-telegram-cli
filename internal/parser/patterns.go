package parser

import "regexp"

// Each pattern captures one named group "val" for the field value.
var (
	// "alertname: HighCPU", "Alert: HighCPU", "alert_name = HighCPU"
	reAlertname = regexp.MustCompile(`(?im)^[ \t]*(?:alertname|alert_name|alert)\s*[:=]\s*(?P<val>\S+)`)

	// "status: firing", "Status: RESOLVED"
	reStatusLabel = regexp.MustCompile(`(?im)^[ \t]*status\s*[:=]\s*(?P<val>\S+)`)

	// standalone keyword on its own line, optionally after emoji/punctuation: "🚨 FIRING"
	reStatusKeyword = regexp.MustCompile(`(?m)^[^a-zA-Z0-9]*(?P<val>FIRING|RESOLVED)\s*$`)

	// "severity = critical", "Severity: critical", severity="critical"
	reSeverity = regexp.MustCompile(`(?im)^[ \t]*severity\s*[:=]["']?\s*(?P<val>critical|warning|info|page|none)["']?`)

	// "host: srv-01", "Host: srv-01"
	reHost = regexp.MustCompile(`(?im)^[ \t]*host\s*[:=]\s*(?P<val>\S+)`)

	// "instance: srv-01:9100"
	reInstance = regexp.MustCompile(`(?im)^[ \t]*instance\s*[:=]\s*(?P<val>\S+)`)

	// "summary: ..." captures rest of line
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
