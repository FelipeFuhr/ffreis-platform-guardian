package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

func TestTableReporter_PrintsHeadersAndTotals(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := &TableReporter{w: &buf}

	rep := &engine.ScanReport{
		RunID:       "run-1",
		GeneratedAt: "2026-01-01T00:00:00Z",
		Results: []engine.RuleResult{
			{Repo: "org/repo", Rule: &rule.Rule{ID: "r1", Name: "R1", Severity: rule.SeverityError}, Status: engine.StatusFail, Message: "bad"},
			{Repo: "org/repo", Rule: &rule.Rule{ID: "r2", Name: "R2", Severity: rule.SeverityWarning}, Status: engine.StatusPass, Message: "ok"},
		},
	}

	if err := r.Report(rep); err != nil {
		t.Fatalf("Report() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "REPO") || !strings.Contains(out, "RULE") || !strings.Contains(out, "SEVERITY") {
		t.Fatalf("expected table header, got:\n%s", out)
	}
	if !strings.Contains(out, "Org totals:") {
		t.Fatalf("expected org totals, got:\n%s", out)
	}
}
