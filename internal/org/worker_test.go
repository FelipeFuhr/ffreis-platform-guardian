package org

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

func TestWorkerPool_ScanAll_AggregatesResults(t *testing.T) {
	reg := rule.NewRegistry()
	_ = reg.AddRule(&rule.Rule{
		ID:       "policy-default-branch",
		Name:     "Default branch must be main",
		Severity: rule.SeverityError,
		Type:     rule.RuleTypePolicy,
		Check: rule.CheckSpec{
			GHRepoSetting: &rule.GHRepoSettingCheck{Field: "default_branch", Value: "main"},
		},
		Remediation: rule.Remediation{Description: "Protect main"},
	})

	eng := engine.NewEngine(reg, zap.NewNop())
	pool := NewWorkerPool(0, eng, zap.NewNop()) // 0 -> default concurrency

	repos := []RepoInfo{
		{FullName: "org/a"},
		{FullName: "org/b"},
	}

	rep, err := pool.ScanAll(context.Background(), repos, "", rule.SeverityError)
	if err != nil {
		t.Fatalf("ScanAll() error = %v", err)
	}

	// Token is empty, so policy rules are filtered; both repos should yield no results.
	if len(rep.Results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(rep.Results))
	}
}
