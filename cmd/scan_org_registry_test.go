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

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = old })

	fn()
	_ = w.Close()

	data, _ := io.ReadAll(r)
	_ = r.Close()
	return string(data)
}

func TestLoadAndValidateRegistry_ErrorsOnInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	if err := os.WriteFile(path, []byte("not: yaml: ["), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := loadAndValidateRegistry([]string{path}, os.Stderr); err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadAndValidateRegistry_Succeeds(t *testing.T) {
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
    description: "Add README"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := loadAndValidateRegistry([]string{path}, os.Stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBaselineOrEmpty(t *testing.T) {
	if baselineOrEmpty(nil) == nil {
		t.Fatalf("expected non-nil")
	}
}

func TestDiscoverRepos_CallsOrgDiscover(t *testing.T) {
	origTransport := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = origTransport })

	http.DefaultClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "/orgs/testorg/repos") {
			t.Fatalf("unexpected URL: %s", req.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`[{"full_name":"testorg/repo1","topics":["t"],"language":"Go","archived":false,"fork":false}]`)),
			Header:     make(http.Header),
		}, nil
	})

	origOrg := scanOrgOrg
	t.Cleanup(func() { scanOrgOrg = origOrg })
	scanOrgOrg = "testorg"

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	stderr := captureStderr(t, func() {
		repos, err := discoverRepos(cmd, "")
		if err != nil {
			t.Fatalf("discoverRepos() error = %v", err)
		}
		if len(repos) != 1 {
			t.Fatalf("expected 1 repo, got %d", len(repos))
		}
	})
	if stderr == "" {
		t.Fatalf("expected stderr output")
	}
}
