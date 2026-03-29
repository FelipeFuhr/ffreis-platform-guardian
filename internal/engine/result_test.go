package engine

import (
	"testing"

	"github.com/ffreis/platform-guardian/internal/rule"
)

func TestScanReport_HasFailures_Threshold(t *testing.T) {
	t.Parallel()

	rep := &ScanReport{
		Results: []RuleResult{
			{Repo: "org/repo", Rule: &rule.Rule{ID: "r1", Severity: rule.SeverityWarning}, Status: StatusFail},
			{Repo: "org/repo", Rule: &rule.Rule{ID: "r2", Severity: rule.SeverityInfo}, Status: StatusFail},
		},
	}

	if rep.HasFailures(rule.SeverityError) {
		t.Fatalf("expected no error-level failures")
	}
	if !rep.HasFailures(rule.SeverityWarning) {
		t.Fatalf("expected warning-level failures")
	}
	if rep.FailureCount() != 2 || rep.PassCount() != 0 || rep.RepoCount() != 1 {
		t.Fatalf("unexpected counts: fail=%d pass=%d repos=%d", rep.FailureCount(), rep.PassCount(), rep.RepoCount())
	}
}

func TestScanReport_Aggregations(t *testing.T) {
	t.Parallel()

	rep := &ScanReport{
		Results: []RuleResult{
			{Repo: "org/a", Rule: &rule.Rule{ID: "r1", Severity: rule.SeverityError}, Status: StatusFail},
			{Repo: "org/a", Rule: &rule.Rule{ID: "r2", Severity: rule.SeverityWarning}, Status: StatusPass},
			{Repo: "org/b", Rule: &rule.Rule{ID: "r1", Severity: rule.SeverityError}, Status: StatusFail},
		},
	}

	rc := rep.RepoSummary()
	if rc["org/a"].Fail != 1 || rc["org/a"].Pass != 1 {
		t.Fatalf("unexpected repo summary for org/a: %+v", rc["org/a"])
	}

	rules := rep.RuleFailureCounts()
	if rules["r1"] != 2 {
		t.Fatalf("expected r1 failures=2, got %d", rules["r1"])
	}

	sev := rep.SeverityBreakdown()
	if sev[string(rule.SeverityError)] != 2 {
		t.Fatalf("expected error breakdown=2, got %d", sev[string(rule.SeverityError)])
	}
}
