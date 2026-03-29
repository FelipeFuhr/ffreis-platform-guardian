package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

func TestSetReportMetadata_SetsRunIDAndTimestamp(t *testing.T) {
	now := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
	rep := &engine.ScanReport{}

	setReportMetadata(rep, now)

	if rep.RunID == "" || rep.GeneratedAt == "" {
		t.Fatalf("expected metadata to be set: %+v", rep)
	}
}

func TestHandleBaseline_WritesBaselineWhenRequested(t *testing.T) {
	dir := t.TempDir()
	readPath := filepath.Join(dir, "missing.json")
	writePath := filepath.Join(dir, "baseline.json")

	origBaseline := scanOrgBaseline
	origWrite := scanOrgWriteBaseline
	t.Cleanup(func() {
		scanOrgBaseline = origBaseline
		scanOrgWriteBaseline = origWrite
	})

	scanOrgBaseline = readPath
	scanOrgWriteBaseline = writePath

	in := &engine.ScanReport{
		RunID:       "run-1",
		GeneratedAt: "2026-01-01T00:00:00Z",
		Results: []engine.RuleResult{
			{Repo: "org/repo", Rule: &rule.Rule{ID: "r1", Severity: rule.SeverityError}, Status: engine.StatusFail, Message: "nope"},
		},
	}

	out, err := handleBaseline(in)
	if err != nil {
		t.Fatalf("handleBaseline() error = %v", err)
	}
	if out == nil {
		t.Fatalf("expected report")
	}
	if _, err := os.Stat(writePath); err != nil {
		t.Fatalf("expected baseline file to be written: %v", err)
	}
}

func TestMaybeWriteJSONReport_WritesFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "report.json")

	orig := scanOrgOutputFile
	t.Cleanup(func() { scanOrgOutputFile = orig })
	scanOrgOutputFile = outPath

	rep := &engine.ScanReport{RunID: "run-1", GeneratedAt: "2026-01-01T00:00:00Z"}
	if err := maybeWriteJSONReport(rep); err != nil {
		t.Fatalf("maybeWriteJSONReport() error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		t.Fatalf("expected non-empty JSON output")
	}
}
