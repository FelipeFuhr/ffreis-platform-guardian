package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/ffreis/platform-guardian/internal/engine"
)

func TestResolveGitHubToken_UsesFlagFirst(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "env-token")
	if got := resolveGitHubToken("flag-token"); got != "flag-token" {
		t.Fatalf("expected flag token, got %q", got)
	}
}

func TestResolveGitHubToken_FallsBackToEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "env-token")
	if got := resolveGitHubToken(""); got != "env-token" {
		t.Fatalf("expected env token, got %q", got)
	}
}

func TestSetReportMetadata(t *testing.T) {
	report := &engine.ScanReport{}
	now := time.Date(2026, 3, 27, 12, 34, 56, 0, time.FixedZone("X", 0))

	setReportMetadata(report, now)

	if report.RunID == "" {
		t.Fatal("expected RunID to be set")
	}
	if report.GeneratedAt != "2026-03-27T12:34:56Z" {
		t.Fatalf("unexpected GeneratedAt: %q", report.GeneratedAt)
	}
}

func TestLoadAndValidateRegistry_InvalidPathErrors(t *testing.T) {
	// Smoke test: passing a non-existent path must error.
	f, err := os.CreateTemp("", "stderr-*")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(f.Name())

	_, err = loadAndValidateRegistry([]string{"./definitely-not-a-real-path"}, f)
	if err == nil {
		t.Fatal("expected error")
	}
}
