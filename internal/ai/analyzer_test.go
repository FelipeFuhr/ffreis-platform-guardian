package ai

import (
	"testing"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

const (
	remediationAddReadme = "Add README.md"
	ruleRequireReadme    = "require-readme"
	repoClean            = "org/clean"
	repoTroubled         = "org/troubled"
)

func makeResult(repo, ruleID string, sev rule.Severity, status engine.CheckStatus, remediation string) engine.RuleResult {
	return engine.RuleResult{
		Repo: repo,
		Rule: &rule.Rule{
			ID:       ruleID,
			Severity: sev,
		},
		Status:      status,
		Remediation: remediation,
	}
}

func TestAnalyzeSystemicViolation(t *testing.T) {
	// Two repos both fail the same rule → systemic violation
	report := &engine.ScanReport{
		Results: []engine.RuleResult{
			makeResult("org/a", ruleRequireReadme, rule.SeverityError, engine.StatusFail, remediationAddReadme),
			makeResult("org/b", ruleRequireReadme, rule.SeverityError, engine.StatusFail, remediationAddReadme),
		},
	}

	analysis := Analyze(report, nil)

	found := false
	for _, p := range analysis.Patterns {
		if p.Kind == "systemic-violation" {
			found = true
		}
	}
	if !found {
		t.Error("expected systemic-violation pattern")
	}
}

func TestAnalyzeCompliantRepo(t *testing.T) {
	report := &engine.ScanReport{
		Results: []engine.RuleResult{
			makeResult(repoClean, ruleRequireReadme, rule.SeverityError, engine.StatusPass, ""),
			makeResult(repoClean, "require-ci", rule.SeverityError, engine.StatusPass, ""),
			makeResult("org/dirty", ruleRequireReadme, rule.SeverityError, engine.StatusFail, remediationAddReadme),
		},
	}

	analysis := Analyze(report, nil)

	found := false
	for _, p := range analysis.Patterns {
		if p.Kind == "fully-compliant" {
			for _, repo := range p.Repos {
				if repo == repoClean {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected clean repo to appear in fully-compliant pattern")
	}
}

func TestAnalyzeWidespreadFailure(t *testing.T) {
	// 9 fail, 1 pass → 90% failure rate
	results := make([]engine.RuleResult, 10)
	for i := 0; i < 9; i++ {
		results[i] = makeResult(repoTroubled, "r"+string(rune('a'+i)), rule.SeverityError, engine.StatusFail, "")
	}
	results[9] = makeResult(repoTroubled, "r-pass", rule.SeverityWarning, engine.StatusPass, "")

	report := &engine.ScanReport{Results: results}
	analysis := Analyze(report, nil)

	found := false
	for _, p := range analysis.Patterns {
		if p.Kind == "widespread-failure" {
			for _, repo := range p.Repos {
				if repo == repoTroubled {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected troubled repo to be flagged as widespread-failure")
	}
}

func TestAnalyzeSuggestionsOrdered(t *testing.T) {
	// require-readme fails in 3 repos, require-ci fails in 1 → readme should be first
	report := &engine.ScanReport{
		Results: []engine.RuleResult{
			makeResult("org/a", ruleRequireReadme, rule.SeverityError, engine.StatusFail, remediationAddReadme),
			makeResult("org/b", ruleRequireReadme, rule.SeverityError, engine.StatusFail, remediationAddReadme),
			makeResult("org/c", ruleRequireReadme, rule.SeverityError, engine.StatusFail, remediationAddReadme),
			makeResult("org/a", "require-ci", rule.SeverityError, engine.StatusFail, "Add CI workflow"),
		},
	}

	analysis := Analyze(report, nil)

	if len(analysis.Suggestions) < 2 {
		t.Fatalf("expected at least 2 suggestions, got %d", len(analysis.Suggestions))
	}
	if analysis.Suggestions[0].RuleID != ruleRequireReadme {
		t.Errorf("expected require-readme first (most repos), got %q", analysis.Suggestions[0].RuleID)
	}
}

func TestAnalyzeEmptyReport(t *testing.T) {
	analysis := Analyze(&engine.ScanReport{}, nil)
	if len(analysis.Patterns) != 0 || len(analysis.Suggestions) != 0 {
		t.Error("expected empty analysis for empty report")
	}
}

func TestFormatAnalysisNoPatterns(t *testing.T) {
	out := FormatAnalysis(&Analysis{})
	if out == "" {
		t.Error("expected non-empty output")
	}
}

func TestCorrelatedRules(t *testing.T) {
	// r1 and r2 always fail together in two repos; r3 fails alone
	report := &engine.ScanReport{
		Results: []engine.RuleResult{
			makeResult("org/a", "r1", rule.SeverityError, engine.StatusFail, ""),
			makeResult("org/a", "r2", rule.SeverityError, engine.StatusFail, ""),
			makeResult("org/b", "r1", rule.SeverityError, engine.StatusFail, ""),
			makeResult("org/b", "r2", rule.SeverityError, engine.StatusFail, ""),
			makeResult("org/c", "r3", rule.SeverityError, engine.StatusFail, ""),
		},
	}

	pairs := correlatedRules(report)

	found := false
	for _, pair := range pairs {
		if (pair[0] == "r1" && pair[1] == "r2") || (pair[0] == "r2" && pair[1] == "r1") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected r1+r2 correlated pair, got %v", pairs)
	}
}
