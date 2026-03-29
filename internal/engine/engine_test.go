package engine

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/ffreis/platform-guardian/internal/rule"
	"github.com/ffreis/platform-guardian/internal/scanner"
)

const (
	testRepo      = "test/repo"
	testRepoScope = "test/my-repo"

	fileReadme   = "README.md"
	fileMakefile = "Makefile"

	testGeneratedAt = "2024-01-01T00:00:00Z"
)

func makeRule(id string, severity rule.Severity, ruleType rule.RuleType, check rule.CheckSpec) *rule.Rule {
	return &rule.Rule{
		ID:       id,
		Name:     id,
		Severity: severity,
		Type:     ruleType,
		Check:    check,
	}
}

func makeRegistry(rules ...*rule.Rule) *rule.Registry {
	reg := rule.NewRegistry()
	for _, r := range rules {
		_ = reg.AddRule(r)
	}
	return reg
}

func TestCheck_AllPass(t *testing.T) {
	// Use an in-memory snapshot with a file that exists
	snap := scanner.NewSnapshot(testRepo)
	snap.FilePaths = []string{fileReadme, fileMakefile}

	r1 := makeRule("r1", rule.SeverityError, rule.RuleTypeStructure, rule.CheckSpec{
		FileExists: &rule.FileExistsCheck{Path: fileReadme},
	})
	r2 := makeRule("r2", rule.SeverityWarning, rule.RuleTypeStructure, rule.CheckSpec{
		FileExists: &rule.FileExistsCheck{Path: fileMakefile},
	})

	reg := makeRegistry(r1, r2)
	log, _ := zap.NewDevelopment()
	eng := NewEngine(reg, log)

	// Evaluate directly (no API calls)
	report := &ScanReport{
		RunID:       "test-run",
		GeneratedAt: testGeneratedAt,
	}

	rules := reg.EffectiveRules(testRepo, nil, nil)
	for _, r := range rules {
		result := Evaluate(r, snap)
		report.Results = append(report.Results, result)
	}

	if report.FailureCount() != 0 {
		t.Errorf("expected 0 failures, got %d", report.FailureCount())
	}
	if report.PassCount() != 2 {
		t.Errorf("expected 2 passes, got %d", report.PassCount())
	}

	_ = eng
}

func TestCheck_OneFails(t *testing.T) {
	snap := scanner.NewSnapshot(testRepo)
	snap.FilePaths = []string{fileReadme} // Makefile absent

	r1 := makeRule("r1", rule.SeverityError, rule.RuleTypeStructure, rule.CheckSpec{
		FileExists: &rule.FileExistsCheck{Path: fileReadme},
	})
	r2 := makeRule("r2", rule.SeverityError, rule.RuleTypeStructure, rule.CheckSpec{
		FileExists: &rule.FileExistsCheck{Path: fileMakefile},
	})

	reg := makeRegistry(r1, r2)

	report := &ScanReport{RunID: "test", GeneratedAt: testGeneratedAt}
	rules := reg.EffectiveRules(testRepo, nil, nil)
	for _, r := range rules {
		result := Evaluate(r, snap)
		report.Results = append(report.Results, result)
	}

	if report.FailureCount() != 1 {
		t.Errorf("expected 1 failure, got %d", report.FailureCount())
	}
	if !report.HasFailures(rule.SeverityError) {
		t.Error("expected HasFailures to be true for error threshold")
	}
}

func TestCheck_ScopeExclusion(t *testing.T) {
	snap := scanner.NewSnapshot(testRepoScope)
	snap.FilePaths = []string{} // no files

	// Rule that is excluded for "test/my-repo"
	r1 := makeRule("r1", rule.SeverityError, rule.RuleTypeStructure, rule.CheckSpec{
		FileExists: &rule.FileExistsCheck{Path: fileReadme},
	})
	r1.Scope.Exclude.Repos = []string{testRepoScope}

	reg := makeRegistry(r1)

	// After EffectiveRules, rule is in the list since EffectiveRules doesn't apply scope
	// But Match should filter it out
	rules := reg.EffectiveRules(testRepoScope, nil, nil)
	matched := Match(testRepoScope, nil, nil, rules)

	if len(matched) != 0 {
		t.Errorf("expected 0 rules after scope exclusion, got %d", len(matched))
	}

	_ = context.Background()
}
