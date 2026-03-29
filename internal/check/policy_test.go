package check

import (
	"testing"

	"github.com/ffreis/platform-guardian/internal/scanner"
)

func TestGHBranchProtectChecker_Evaluate(t *testing.T) {
	t.Parallel()

	snap := scanner.NewSnapshot("org/repo")
	snap.BranchProtection["main"] = scanner.BranchProtection{
		RequirePRReviews:    true,
		RequireStatusChecks: true,
	}

	c := &GHBranchProtectChecker{
		Branch:              "main",
		RequirePRReviews:    true,
		RequireStatusChecks: true,
	}

	got := c.Evaluate(snap)
	if got.Status != Pass {
		t.Fatalf("expected Pass, got %s: %s", got.Status, got.Message)
	}
}

func TestGHBranchProtectChecker_MissingBranchProtectionFails(t *testing.T) {
	t.Parallel()

	snap := scanner.NewSnapshot("org/repo")
	c := &GHBranchProtectChecker{Branch: "main"}

	got := c.Evaluate(snap)
	if got.Status != Fail {
		t.Fatalf("expected Fail, got %s: %s", got.Status, got.Message)
	}
}

func TestGHTeamPermissionChecker_Evaluate(t *testing.T) {
	t.Parallel()

	snap := scanner.NewSnapshot("org/repo")
	snap.TeamPermissions["platform"] = scanner.TeamPermission{Permission: "write"}

	c := &GHTeamPermissionChecker{Team: "platform", Permission: "triage"}
	got := c.Evaluate(snap)
	if got.Status != Pass {
		t.Fatalf("expected Pass, got %s: %s", got.Status, got.Message)
	}
}

func TestGHRepoSettingChecker_UnknownFieldErrors(t *testing.T) {
	t.Parallel()

	snap := scanner.NewSnapshot("org/repo")
	c := &GHRepoSettingChecker{Field: "unknown", Value: "x"}

	got := c.Evaluate(snap)
	if got.Status != Error {
		t.Fatalf("expected Error, got %s: %s", got.Status, got.Message)
	}
}

func TestGHRepoSettingChecker_DefaultBranchPass(t *testing.T) {
	t.Parallel()

	snap := scanner.NewSnapshot("org/repo")
	snap.Settings.DefaultBranch = "main"

	c := &GHRepoSettingChecker{Field: "default_branch", Value: "main"}
	got := c.Evaluate(snap)
	if got.Status != Pass {
		t.Fatalf("expected Pass, got %s: %s", got.Status, got.Message)
	}
}
