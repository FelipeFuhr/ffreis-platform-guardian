package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ffreis/platform-guardian/internal/engine"
)

func TestJSONReporter_EncodesReport(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := &JSONReporter{w: &buf}

	rep := &engine.ScanReport{RunID: "run-1", GeneratedAt: "2026-01-01T00:00:00Z"}
	if err := r.Report(rep); err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if !strings.Contains(buf.String(), "\"RunID\"") {
		t.Fatalf("expected JSON output to contain RunID, got:\n%s", buf.String())
	}
}
