package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunValidate_AllValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")

	content := `apiVersion: guardian/v1
kind: Rule
metadata:
  id: test-rule-1
  name: Test Rule
  severity: error
spec:
  type: structure
  check:
    file_exists:
      path: README.md
  remediation:
    description: "Add README.md"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write rule file: %v", err)
	}

	validateRules = []string{path}
	t.Cleanup(func() { validateRules = nil })

	if err := runValidate(&cobra.Command{}, nil); err != nil {
		t.Fatalf("runValidate() error = %v", err)
	}
}

func TestRunValidate_InvalidRuleErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")

	// Missing required metadata.id and spec fields -> should fail validation.
	content := `apiVersion: guardian/v1
kind: Rule
metadata:
  name: Missing ID
spec:
  type: structure
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write rule file: %v", err)
	}

	validateRules = []string{path}
	t.Cleanup(func() { validateRules = nil })

	if err := runValidate(&cobra.Command{}, nil); err == nil {
		t.Fatalf("expected validation error")
	}
}
