package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

func TestSummaryReporter_WithFailures_PrintsBreakdowns(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := &SummaryReporter{w: &buf}

	rep := &engine.ScanReport{
		RunID:       "run-1",
		GeneratedAt: "2026-01-01T00:00:00Z",
		Results: []engine.RuleResult{
			{Repo: "org/a", Rule: &rule.Rule{ID: "r1", Name: "R1", Severity: rule.SeverityError}, Status: engine.StatusFail, Message: "bad"},
			{Repo: "org/b", Rule: &rule.Rule{ID: "r1", Name: "R1", Severity: rule.SeverityError}, Status: engine.StatusFail, Message: "bad"},
			{Repo: "org/b", Rule: &rule.Rule{ID: "r2", Name: "R2", Severity: rule.SeverityWarning}, Status: engine.StatusPass, Message: "ok"},
		},
	}

	if err := r.Report(rep); err != nil {
		t.Fatalf("Report() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Failures by severity:") {
		t.Fatalf("expected severity breakdown, got:\n%s", out)
	}
	if !strings.Contains(out, "Most violated rules:") {
		t.Fatalf("expected most violated rules section, got:\n%s", out)
	}
	if !strings.Contains(out, "Repos with failures") {
		t.Fatalf("expected failing repos section, got:\n%s", out)
	}
}
