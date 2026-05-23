package scanner

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPolicyScanner_SetsSettingsAndPermissions(t *testing.T) {
	origTransport := HTTPClient.Transport
	t.Cleanup(func() { HTTPClient.Transport = origTransport })

	snap := NewSnapshot("org/repo")
	s := NewPolicyScanner(snap, io.Discard)

	HTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get(httpHeaderAuthorization) != authBearerPrefix+"token" {
			t.Fatalf("expected auth header to be set")
		}
		switch {
		case req.URL.Path == "/repos/org/repo":
			return httpResponse(http.StatusOK, `{"default_branch":"main","private":true,"allow_squash_merge":true,"allow_merge_commit":false,"allow_rebase_merge":false}`), nil
		case strings.HasPrefix(req.URL.Path, "/repos/org/repo/branches/") && strings.HasSuffix(req.URL.Path, "/protection"):
			// Simulate unprotected branch.
			return httpResponse(http.StatusNotFound, ""), nil
		case req.URL.Path == "/orgs/org/teams":
			return httpResponse(http.StatusOK, `[{"slug":"platform"}]`), nil
		case strings.HasPrefix(req.URL.Path, "/orgs/org/teams/platform/repos/"):
			return httpResponse(http.StatusOK, `{"permissions":{"admin":false,"maintain":false,"push":true,"triage":false,"pull":true}}`), nil
		default:
			return httpResponse(http.StatusNotFound, ""), nil
		}
	})

	if err := s.Scan(context.Background(), "token", "org/repo"); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if snap.Settings.DefaultBranch != "main" || !snap.Settings.Private {
		t.Fatalf("expected settings to be populated, got %+v", snap.Settings)
	}
	if _, ok := snap.BranchProtection["main"]; !ok {
		t.Fatalf("expected branch protection to be recorded for default branch")
	}
	if got := snap.TeamPermissions["platform"].Permission; got != "write" {
		t.Fatalf("expected team permission write, got %q", got)
	}
}

func TestPolicyScanner_FetchRepoSettings_ErrorsOnNon200(t *testing.T) {
	origTransport := HTTPClient.Transport
	t.Cleanup(func() { HTTPClient.Transport = origTransport })

	snap := NewSnapshot("org/repo")
	s := NewPolicyScanner(snap, io.Discard)

	HTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})

	if err := s.Scan(context.Background(), "token", "org/repo"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPolicyScanner_WritesWarningsToConfiguredWriter(t *testing.T) {
	origTransport := HTTPClient.Transport
	t.Cleanup(func() { HTTPClient.Transport = origTransport })

	snap := NewSnapshot("org/repo")
	var warnings bytes.Buffer
	s := NewPolicyScanner(snap, &warnings)

	HTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Path == "/repos/org/repo":
			return httpResponse(http.StatusOK, `{"default_branch":"main","private":true,"allow_squash_merge":true,"allow_merge_commit":false,"allow_rebase_merge":false}`), nil
		case strings.HasPrefix(req.URL.Path, "/repos/org/repo/branches/") && strings.HasSuffix(req.URL.Path, "/protection"):
			return httpResponse(http.StatusForbidden, ""), nil
		case req.URL.Path == "/orgs/org/teams":
			return httpResponse(http.StatusForbidden, ""), nil
		default:
			return httpResponse(http.StatusNotFound, ""), nil
		}
	})

	if err := s.Scan(context.Background(), "token", "org/repo"); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	output := warnings.String()
	if !strings.Contains(output, "could not fetch branch protection") {
		t.Fatalf("expected branch protection warning, got %q", output)
	}
	if !strings.Contains(output, "could not fetch team permissions") {
		t.Fatalf("expected team permissions warning, got %q", output)
	}
}
