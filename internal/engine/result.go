package engine

import (
	"github.com/ffreis/platform-guardian/internal/rule"
)

type CheckStatus string

const (
	StatusPass  CheckStatus = "pass"
	StatusFail  CheckStatus = "fail"
	StatusSkip  CheckStatus = "skip"
	StatusError CheckStatus = "error"
)

type RuleResult struct {
	Repo        string
	Rule        *rule.Rule
	Status      CheckStatus
	Message     string
	Evidence    []string
	Remediation string
}

type ScanReport struct {
	RunID       string
	GeneratedAt string // RFC3339
	Results     []RuleResult
}

// HasFailures returns true if any result has StatusFail and its rule severity >= threshold.
func (r *ScanReport) HasFailures(threshold rule.Severity) bool {
	threshLevel := severityLevel(threshold)
	for _, result := range r.Results {
		if result.Status == StatusFail {
			if severityLevel(result.Rule.Severity) >= threshLevel {
				return true
			}
		}
	}
	return false
}

func (r *ScanReport) FailureCount() int {
	count := 0
	for _, result := range r.Results {
		if result.Status == StatusFail {
			count++
		}
	}
	return count
}

func (r *ScanReport) PassCount() int {
	count := 0
	for _, result := range r.Results {
		if result.Status == StatusPass {
			count++
		}
	}
	return count
}

func severityLevel(s rule.Severity) int {
	switch s {
	case rule.SeverityInfo:
		return 1
	case rule.SeverityWarning:
		return 2
	case rule.SeverityError:
		return 3
	}
	return 0
}

// RepoStats holds per-repository scan counts.
type RepoStats struct {
	Repo  string
	Pass  int
	Fail  int
	Skip  int
	Error int
}

// RepoSummary returns per-repo pass/fail/skip/error counts.
func (r *ScanReport) RepoSummary() map[string]*RepoStats {
	stats := map[string]*RepoStats{}
	for _, result := range r.Results {
		s, ok := stats[result.Repo]
		if !ok {
			s = &RepoStats{Repo: result.Repo}
			stats[result.Repo] = s
		}
		switch result.Status {
		case StatusPass:
			s.Pass++
		case StatusFail:
			s.Fail++
		case StatusSkip:
			s.Skip++
		case StatusError:
			s.Error++
		}
	}
	return stats
}

// RuleFailureCounts returns ruleID → failure count across all repos.
func (r *ScanReport) RuleFailureCounts() map[string]int {
	counts := map[string]int{}
	for _, result := range r.Results {
		if result.Status == StatusFail && result.Rule != nil {
			counts[result.Rule.ID]++
		}
	}
	return counts
}

// SeverityBreakdown returns failure counts grouped by severity.
func (r *ScanReport) SeverityBreakdown() map[string]int {
	breakdown := map[string]int{}
	for _, result := range r.Results {
		if result.Status == StatusFail && result.Rule != nil {
			breakdown[string(result.Rule.Severity)]++
		}
	}
	return breakdown
}

// RepoCount returns the number of distinct repos in the report.
func (r *ScanReport) RepoCount() int {
	seen := map[string]bool{}
	for _, result := range r.Results {
		seen[result.Repo] = true
	}
	return len(seen)
}
