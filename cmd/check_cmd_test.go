package cmd

import (
	"os"
	"path/filepath"
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
