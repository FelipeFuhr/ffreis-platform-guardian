package check

import (
	"testing"

	"github.com/ffreis/platform-guardian/internal/scanner"
)

func TestPermissionLevel(t *testing.T) {
	t.Parallel()

	if permissionLevel("pull") != 1 || permissionLevel("read") != 1 {
		t.Fatalf("unexpected read/pull level")
	}
	if permissionLevel("push") != 3 || permissionLevel("write") != 3 {
		t.Fatalf("unexpected write/push level")
	}
	if permissionLevel("admin") != 5 {
		t.Fatalf("unexpected admin level")
	}
}

func TestGHTeamPermissionChecker_FailsWhenInsufficient(t *testing.T) {
	t.Parallel()

	snap := scanner.NewSnapshot("org/repo")
	snap.TeamPermissions["platform"] = scanner.TeamPermission{Permission: "read"}

	c := &GHTeamPermissionChecker{Team: "platform", Permission: "write"}
	got := c.Evaluate(snap)
	if got.Status != Fail {
		t.Fatalf("expected Fail, got %s: %s", got.Status, got.Message)
	}
}

func TestGHRepoSettingChecker_BoolFields(t *testing.T) {
	t.Parallel()

	snap := scanner.NewSnapshot("org/repo")
	snap.Settings.AllowSquashMerge = true
	snap.Settings.AllowMergeCommit = false
	snap.Settings.AllowRebaseMerge = false
	snap.Settings.Private = true

	tests := []struct {
		field string
		value string
	}{
		{field: "allow_squash_merge", value: "true"},
		{field: "allow_merge_commit", value: "false"},
		{field: "allow_rebase_merge", value: "false"},
		{field: "private", value: "true"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.field, func(t *testing.T) {
			c := &GHRepoSettingChecker{Field: tc.field, Value: tc.value}
			got := c.Evaluate(snap)
			if got.Status != Pass {
				t.Fatalf("expected Pass, got %s: %s", got.Status, got.Message)
			}
		})
	}
}
