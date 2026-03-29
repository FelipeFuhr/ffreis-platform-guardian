package baseline

import (
	"testing"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

func makeResult(repo, ruleID string, status engine.CheckStatus) engine.RuleResult {
	return engine.RuleResult{
		Repo: repo,
		Rule: &rule.Rule{
			ID:       ruleID,
			Severity: rule.SeverityError,
		},
		Status: status,
	}
}

func makeReport(results ...engine.RuleResult) *engine.ScanReport {
	return &engine.ScanReport{
		RunID:       "test-run",
		GeneratedAt: "2024-01-01T00:00:00Z",
		Results:     results,
	}
}

func TestFilterNew_NewFailure(t *testing.T) {
	b := &Baseline{Entries: []Entry{}}
	report := makeReport(makeResult("org/repo", "rule-1", engine.StatusFail))

	filtered := FilterNew(report, b)
	if len(filtered.Results) != 1 {
		t.Errorf("expected 1 new result, got %d", len(filtered.Results))
	}
}

func TestFilterNew_ExistingFailure(t *testing.T) {
	b := &Baseline{
		Entries: []Entry{
			{Repo: "org/repo", RuleID: "rule-1", Status: "fail", FirstSeen: "2024-01-01T00:00:00Z"},
		},
	}
	report := makeReport(makeResult("org/repo", "rule-1", engine.StatusFail))

	filtered := FilterNew(report, b)
	if len(filtered.Results) != 0 {
		t.Errorf("expected 0 new results (already in baseline), got %d", len(filtered.Results))
	}
}

func TestFilterNew_ResolvedPass(t *testing.T) {
	b := &Baseline{
		Entries: []Entry{
			{Repo: "org/repo", RuleID: "rule-1", Status: "fail", FirstSeen: "2024-01-01T00:00:00Z"},
		},
	}
	// Now it passes
	report := makeReport(makeResult("org/repo", "rule-1", engine.StatusPass))

	filtered := FilterNew(report, b)
	// Pass results are included in output (they're not new failures)
	// But they should not be in the filtered output as failures
	for _, r := range filtered.Results {
		if r.Status == engine.StatusFail {
			t.Error("expected no failures in filtered output for resolved rule")
		}
	}
}

func TestUpdate_AddsNewEntry(t *testing.T) {
	b := &Baseline{Entries: []Entry{}}
	report := makeReport(makeResult("org/repo", "rule-1", engine.StatusFail))

	updated := Update(b, report)
	if len(updated.Entries) != 1 {
		t.Errorf("expected 1 entry in updated baseline, got %d", len(updated.Entries))
	}
	if updated.Entries[0].RuleID != "rule-1" {
		t.Errorf("expected rule-1, got %s", updated.Entries[0].RuleID)
	}
}

func TestUpdate_RemovesResolvedEntry(t *testing.T) {
	b := &Baseline{
		Entries: []Entry{
			{Repo: "org/repo", RuleID: "rule-1", Status: "fail", FirstSeen: "2024-01-01T00:00:00Z"},
		},
	}
	// Rule now passes
	report := makeReport(makeResult("org/repo", "rule-1", engine.StatusPass))

	updated := Update(b, report)
	if len(updated.Entries) != 0 {
		t.Errorf("expected 0 entries after resolution, got %d", len(updated.Entries))
	}
}
