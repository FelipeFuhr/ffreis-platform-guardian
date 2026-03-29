package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

func TestSummaryReporter_AllPassesPrintsSuccessLine(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := &SummaryReporter{w: &buf}

	rep := &engine.ScanReport{
		RunID:       "run-1",
		GeneratedAt: "2026-01-01T00:00:00Z",
		Results: []engine.RuleResult{
			{Repo: "org/one", Rule: &rule.Rule{ID: "r1", Name: "R1", Severity: rule.SeverityError}, Status: engine.StatusPass, Message: "ok"},
			{Repo: "org/two", Rule: &rule.Rule{ID: "r1", Name: "R1", Severity: rule.SeverityError}, Status: engine.StatusPass, Message: "ok"},
		},
	}

	if err := r.Report(rep); err != nil {
		t.Fatalf("Report() error = %v", err)
	}

	if !strings.Contains(buf.String(), "All checks passed.") {
		t.Fatalf("expected success line, got:\n%s", buf.String())
	}
}
