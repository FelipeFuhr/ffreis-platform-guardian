package cmd

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunScanOrg_PolicyOnlyNoToken_DoesNotExit(t *testing.T) {
	origTransport := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = origTransport })

	http.DefaultClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// Only org discovery should hit the network in this scenario.
		if !strings.Contains(req.URL.Path, "/orgs/testorg/repos") {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
				Header:     make(http.Header),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`[{"full_name":"testorg/repo1","topics":[],"language":"Go","archived":false,"fork":false}]`)),
			Header:     make(http.Header),
		}, nil
	})

	dir := t.TempDir()
	rulesPath := filepath.Join(dir, "rules.yaml")
	policyOnlyRule := `apiVersion: guardian/v1
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
    description: "Set default branch to main"
`
	if err := os.WriteFile(rulesPath, []byte(policyOnlyRule), 0o644); err != nil {
		t.Fatalf("write rule file: %v", err)
	}

	// Save and restore global flags used by cmd/scan_org.go
	origOrg := scanOrgOrg
	origRules := scanOrgRules
	origToken := scanOrgToken
	origFormat := scanOrgFormat
	origFailOn := scanOrgFailOn
	origConcurrency := scanOrgConcurrency
	origBaseline := scanOrgBaseline
	origWriteBaseline := scanOrgWriteBaseline
	origOutput := scanOrgOutputFile
	origAI := scanOrgAISuggest

	t.Cleanup(func() {
		scanOrgOrg = origOrg
		scanOrgRules = origRules
		scanOrgToken = origToken
		scanOrgFormat = origFormat
		scanOrgFailOn = origFailOn
		scanOrgConcurrency = origConcurrency
		scanOrgBaseline = origBaseline
		scanOrgWriteBaseline = origWriteBaseline
		scanOrgOutputFile = origOutput
		scanOrgAISuggest = origAI
	})

	scanOrgOrg = "testorg"
	scanOrgRules = []string{rulesPath}
	scanOrgToken = "" // no token -> policy rules filtered -> no failures -> no os.Exit
	scanOrgFormat = "summary"
	scanOrgFailOn = "error"
	scanOrgConcurrency = 1
	scanOrgBaseline = ""
	scanOrgWriteBaseline = ""
	scanOrgOutputFile = ""
	scanOrgAISuggest = false

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// Silence output to avoid noisy test logs.
	_ = captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := runScanOrg(cmd, nil); err != nil {
				t.Fatalf("runScanOrg() error = %v", err)
			}
		})
	})
}
