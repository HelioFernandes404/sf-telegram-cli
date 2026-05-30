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

// Parse tolerantly extracts alert fields from raw Telegram message text.
func Parse(text string) Result {
	var r Result
	r.MatchedPatterns = []string{}
	r.MissingFields = []string{}

	grab := func(target *string, patternName, val string) {
		if val != "" && *target == "" {
			*target = val
			r.MatchedPatterns = append(r.MatchedPatterns, patternName)
		}
	}

	grab(&r.Fields.Alertname, "alertname_label", namedMatch(reAlertname, text))
	grab(&r.Fields.Host, "host_candidate", namedMatch(reHost, text))
	grab(&r.Fields.Instance, "instance_label", namedMatch(reInstance, text))
	grab(&r.Fields.Job, "job_label", namedMatch(reJob, text))
	grab(&r.Fields.Environment, "environment_label", namedMatch(reEnvironment, text))
	grab(&r.Fields.Namespace, "namespace_label", namedMatch(reNamespace, text))
	grab(&r.Fields.Summary, "summary_label", namedMatch(reSummary, text))
	grab(&r.Fields.Description, "description_label", namedMatch(reDescription, text))

	// status: try label first, then keyword; normalize to lowercase
	if raw := namedMatch(reStatusLabel, text); raw != "" {
		r.Fields.Status = strings.ToLower(raw)
		r.MatchedPatterns = append(r.MatchedPatterns, "status_keyword")
	} else if raw := namedMatch(reStatusKeyword, text); raw != "" {
		r.Fields.Status = strings.ToLower(raw)
		r.MatchedPatterns = append(r.MatchedPatterns, "status_keyword")
	}

	// severity: normalize to lowercase
	if raw := namedMatch(reSeverity, text); raw != "" {
		r.Fields.Severity = strings.ToLower(raw)
		r.MatchedPatterns = append(r.MatchedPatterns, "severity_label")
	}

	r.MissingFields = missingFields(r.Fields)
	r.Confidence = confidence(r.Fields)
	return r
}

func missingFields(f Fields) []string {
	missing := []string{}
	checks := []struct {
		name, val string
	}{
		{"alertname", f.Alertname},
		{"status", f.Status},
		{"severity", f.Severity},
		{"host", f.Host},
		{"instance", f.Instance},
	}
	for _, c := range checks {
		if c.val == "" {
			missing = append(missing, c.name)
		}
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
