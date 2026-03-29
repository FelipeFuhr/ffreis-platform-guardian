package baseline

import (
	"testing"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

const (
	testGeneratedAt = "2024-01-01T00:00:00Z"
	testRepo        = "org/repo"
	testRuleID      = "rule-1"
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
		GeneratedAt: testGeneratedAt,
		Results:     results,
	}
}

func TestFilterNewNewFailure(t *testing.T) {
	b := &Baseline{Entries: []Entry{}}
	report := makeReport(makeResult(testRepo, testRuleID, engine.StatusFail))

	filtered := FilterNew(report, b)
	if len(filtered.Results) != 1 {
		t.Errorf("expected 1 new result, got %d", len(filtered.Results))
	}
}

func TestFilterNewExistingFailure(t *testing.T) {
	b := &Baseline{
		Entries: []Entry{
			{Repo: testRepo, RuleID: testRuleID, Status: "fail", FirstSeen: testGeneratedAt},
		},
	}
	report := makeReport(makeResult(testRepo, testRuleID, engine.StatusFail))

	filtered := FilterNew(report, b)
	if len(filtered.Results) != 0 {
		t.Errorf("expected 0 new results (already in baseline), got %d", len(filtered.Results))
	}
}

func TestFilterNewResolvedPass(t *testing.T) {
	b := &Baseline{
		Entries: []Entry{
			{Repo: testRepo, RuleID: testRuleID, Status: "fail", FirstSeen: testGeneratedAt},
		},
	}
	// Now it passes
	report := makeReport(makeResult(testRepo, testRuleID, engine.StatusPass))

	filtered := FilterNew(report, b)
	// Pass results are included in output (they're not new failures)
	// But they should not be in the filtered output as failures
	for _, r := range filtered.Results {
		if r.Status == engine.StatusFail {
			t.Error("expected no failures in filtered output for resolved rule")
		}
	}
}

func TestUpdateAddsNewEntry(t *testing.T) {
	b := &Baseline{Entries: []Entry{}}
	report := makeReport(makeResult(testRepo, testRuleID, engine.StatusFail))

	updated := Update(b, report)
	if len(updated.Entries) != 1 {
		t.Errorf("expected 1 entry in updated baseline, got %d", len(updated.Entries))
	}
	if updated.Entries[0].RuleID != testRuleID {
		t.Errorf("expected %s, got %s", testRuleID, updated.Entries[0].RuleID)
	}
}

func TestUpdateRemovesResolvedEntry(t *testing.T) {
	b := &Baseline{
		Entries: []Entry{
			{Repo: testRepo, RuleID: testRuleID, Status: "fail", FirstSeen: testGeneratedAt},
		},
	}
	// Rule now passes
	report := makeReport(makeResult(testRepo, testRuleID, engine.StatusPass))

	updated := Update(b, report)
	if len(updated.Entries) != 0 {
		t.Errorf("expected 0 entries after resolution, got %d", len(updated.Entries))
	}
}
