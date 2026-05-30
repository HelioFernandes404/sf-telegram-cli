package search

import "time"

// Request is the normalised set of search parameters.
type Request struct {
	Channel   string
	Query     string
	Since     *time.Time
	Until     *time.Time
	Host      string
	Instance  string
	Alertname string
	Severity  string
	Status    string
	Limit     int
	PageSize  int
	MaxScan   int
	Timeout   time.Duration
	Format    string
	PageToken string
}
