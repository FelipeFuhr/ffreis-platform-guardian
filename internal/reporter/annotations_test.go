package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

func TestAnnotationsReporter_EmitsOnlyFailures(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := &AnnotationsReporter{w: &buf}

	report := &engine.ScanReport{
		RunID:       "run-1",
		GeneratedAt: "2026-01-01T00:00:00Z",
		Results: []engine.RuleResult{
			{Repo: "org/repo", Rule: &rule.Rule{ID: "r1", Name: "rule-1", Severity: rule.SeverityError}, Status: engine.StatusFail, Message: "boom"},
			{Repo: "org/repo", Rule: &rule.Rule{ID: "r2", Name: "rule-2", Severity: rule.SeverityWarning}, Status: engine.StatusFail, Message: "warn"},
			{Repo: "org/repo", Rule: &rule.Rule{ID: "r3", Name: "rule-3", Severity: rule.SeverityInfo}, Status: engine.StatusFail, Message: "info"},
			{Repo: "org/repo", Rule: &rule.Rule{ID: "r4", Name: "rule-4", Severity: rule.SeverityError}, Status: engine.StatusPass, Message: "ok"},
		},
	}

	if err := r.Report(report); err != nil {
		t.Fatalf("Report() error = %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "rule-4") {
		t.Fatalf("expected pass result to be omitted; got output:\n%s", out)
	}
	if !strings.Contains(out, "::error title=rule-1::boom [org/repo]") {
		t.Fatalf("expected error annotation; got output:\n%s", out)
	}
	if !strings.Contains(out, "::warning title=rule-2::warn [org/repo]") {
		t.Fatalf("expected warning annotation; got output:\n%s", out)
	}
	if !strings.Contains(out, "::notice title=rule-3::info [org/repo]") {
		t.Fatalf("expected notice annotation; got output:\n%s", out)
	}
}
