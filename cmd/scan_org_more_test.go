package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/org"
	"github.com/ffreis/platform-guardian/internal/rule"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })

	fn()
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	_ = r.Close()
	return buf.String()
}

func TestScanRepos_WithPolicyOnlyAndNoToken_ReturnsEmptyReport(t *testing.T) {
	reg := rule.NewRegistry()
	_ = reg.AddRule(&rule.Rule{
		ID:       "policy-default-branch",
		Name:     "Default branch must be main",
		Severity: rule.SeverityError,
		Type:     rule.RuleTypePolicy,
		Check: rule.CheckSpec{
			GHRepoSetting: &rule.GHRepoSettingCheck{Field: "default_branch", Value: "main"},
		},
	})

	repos := []org.RepoInfo{{FullName: "org/a"}, {FullName: "org/b"}}
	report, _, err := scanRepos(&cobra.Command{}, reg, zap.NewNop(), repos, "")
	if err != nil {
		t.Fatalf("scanRepos() error = %v", err)
	}
	if len(report.Results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(report.Results))
	}
}

func TestReportToStdout_PrintsSomething(t *testing.T) {
	orig := scanOrgFormat
	t.Cleanup(func() { scanOrgFormat = orig })
	scanOrgFormat = "summary"

	rep := &engine.ScanReport{
		RunID:       "run-1",
		GeneratedAt: "2026-01-01T00:00:00Z",
		Results:     nil,
	}

	out := captureStdout(t, func() {
		if err := reportToStdout(rep); err != nil {
			t.Fatalf("reportToStdout() error = %v", err)
		}
	})
	if out == "" {
		t.Fatalf("expected output")
	}
}

func TestMaybeRunAIAnalysis_WritesAnalysis(t *testing.T) {
	orig := scanOrgAISuggest
	t.Cleanup(func() { scanOrgAISuggest = orig })
	scanOrgAISuggest = true

	rep := &engine.ScanReport{
		RunID:       "run-1",
		GeneratedAt: "2026-01-01T00:00:00Z",
		Results: []engine.RuleResult{
			{Repo: "org/a", Rule: &rule.Rule{ID: "require-readme", Severity: rule.SeverityError}, Status: engine.StatusFail, Message: "missing"},
			{Repo: "org/b", Rule: &rule.Rule{ID: "require-readme", Severity: rule.SeverityError}, Status: engine.StatusFail, Message: "missing"},
		},
	}

	out := captureStdout(t, func() {
		maybeRunAIAnalysis(&cobra.Command{}, rep)
	})
	if out == "" {
		t.Fatalf("expected analysis output")
	}
}
