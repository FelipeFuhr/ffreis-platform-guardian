package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunCheck_PolicyRulesFilteredWithoutToken_DoesNotExit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")

	content := `apiVersion: guardian/v1
kind: Rule
metadata:
  id: policy-default-branch
  name: Default branch must be main
  severity: error
spec:
  type: policy
  check:
    gh_repo_setting:
      field: default_branch
      value: main
  remediation:
    description: "Protect main"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write rule file: %v", err)
	}

	checkRepo = "org/repo"
	checkRules = []string{path}
	checkToken = ""
	checkRef = ""
	checkFormat = "table"
	checkFailOn = "error"
	t.Cleanup(func() {
		checkRepo = ""
		checkRules = nil
		checkToken = ""
		checkRef = ""
		checkFormat = "table"
		checkFailOn = "error"
	})

	if err := runCheck(&cobra.Command{}, nil); err != nil {
		t.Fatalf("runCheck() error = %v", err)
	}
}

// TestRunCheck_SarifFormat_StdoutIsSingleValidJSONDocument is a regression
// test for a bug where a successful `check --format sarif` run wrote a
// trailing "[ok] check completed..." status line to stdout *after* the SARIF
// reporter had already written a complete JSON document there. Any consumer
// redirecting stdout to a file (e.g. CI's `> guardian-report.sarif`) got a
// file with valid JSON followed by garbage, which github/codeql-action's
// upload-sarif step rejected with "Invalid SARIF. JSON syntax error:
// Unexpected non-whitespace character after JSON". Machine-readable formats
// (sarif, json, annotations) must have stdout contain *only* the report;
// status/log text belongs on stderr, regardless of pass/fail outcome.
func TestRunCheck_SarifFormat_StdoutIsSingleValidJSONDocument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")

	// A policy rule with no token configured is filtered out entirely by the
	// engine before evaluation, so the report has zero results (zero
	// failures) and this test needs no network access to reach the success
	// path where the bug lived.
	content := `apiVersion: guardian/v1
kind: Rule
metadata:
  id: policy-default-branch
  name: Default branch must be main
  severity: error
spec:
  type: policy
  check:
    gh_repo_setting:
      field: default_branch
      value: main
  remediation:
    description: "Protect main"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write rule file: %v", err)
	}

	checkRepo = "org/repo"
	checkRules = []string{path}
	checkToken = ""
	checkRef = ""
	checkFormat = "sarif"
	checkFailOn = "error"
	t.Cleanup(func() {
		checkRepo = ""
		checkRules = nil
		checkToken = ""
		checkRef = ""
		checkFormat = "table"
		checkFailOn = "error"
	})

	stdout := captureStdout(t, func() {
		if err := runCheck(&cobra.Command{}, nil); err != nil {
			t.Fatalf("runCheck() error = %v", err)
		}
	})

	if strings.TrimSpace(stdout) == "" {
		t.Fatalf("expected SARIF output on stdout, got empty string")
	}
	if !json.Valid([]byte(stdout)) {
		t.Fatalf("stdout is not a single valid JSON document (trailing content appended after the report?): %q", stdout)
	}
	if strings.Contains(stdout, "check completed") {
		t.Fatalf("status line leaked into stdout report output: %q", stdout)
	}
}
