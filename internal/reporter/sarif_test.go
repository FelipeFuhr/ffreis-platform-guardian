package reporter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

func TestSARIFReporter_ProducesValidShape(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := &SARIFReporter{w: &buf}

	rep := &engine.ScanReport{
		RunID:       "run-1",
		GeneratedAt: "2026-01-01T00:00:00Z",
		Results: []engine.RuleResult{
			{
				Repo:    "org/repo",
				Rule:    &rule.Rule{ID: "rule-1", Name: "Rule One", Severity: rule.SeverityError, Tags: []string{"tag"}},
				Status:  engine.StatusFail,
				Message: "failure",
			},
			{
				Repo:    "org/repo",
				Rule:    &rule.Rule{ID: "rule-1", Name: "Rule One", Severity: rule.SeverityError, Tags: []string{"tag"}},
				Status:  engine.StatusPass,
				Message: "ok",
			},
		},
	}

	if err := r.Report(rep); err != nil {
		t.Fatalf("Report() error = %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	runs, ok := out["runs"].([]any)
	if !ok || len(runs) != 1 {
		t.Fatalf("expected runs to be array with 1 item, got %#v", out["runs"])
	}
	run, _ := runs[0].(map[string]any)
	results, ok := run["results"].([]any)
	if !ok {
		t.Fatalf("expected runs[0].results to be an array")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 SARIF result (failures only), got %d", len(results))
	}
}
