package check

import (
	"fmt"
	"strconv"

	"github.com/ffreis/platform-guardian/internal/scanner"
)

// GHBranchProtectChecker checks branch protection settings.
type GHBranchProtectChecker struct {
	Branch              string
	RequirePRReviews    bool
	RequireStatusChecks bool
}

func (c *GHBranchProtectChecker) Evaluate(snap *scanner.RepoSnapshot) Result {
	bp, ok := snap.BranchProtection[c.Branch]
	if !ok {
		return Result{
			Status:  Fail,
			Message: fmt.Sprintf("no branch protection found for branch %q", c.Branch),
		}
	}

	var issues []string
	if c.RequirePRReviews && !bp.RequirePRReviews {
		issues = append(issues, "require_pr_reviews not enabled")
	}
	if c.RequireStatusChecks && !bp.RequireStatusChecks {
		issues = append(issues, "require_status_checks not enabled")
	}

	if len(issues) > 0 {
		return Result{
			Status:   Fail,
			Message:  fmt.Sprintf("branch protection for %q does not meet requirements", c.Branch),
			Evidence: issues,
		}
	}

	return Result{
		Status:  Pass,
		Message: fmt.Sprintf("branch protection for %q meets requirements", c.Branch),
	}
}

// permissionLevel returns a numeric level for comparison.
func permissionLevel(perm string) int {
	switch perm {
	case "read", "pull":
		return 1
	case "triage":
		return 2
	case "write", "push":
		return 3
	case "maintain":
		return 4
	case "admin":
		return 5
	}
	return 0
}

// GHTeamPermissionChecker checks team permission level.
type GHTeamPermissionChecker struct {
	Team       string
	Permission string
}

func (c *GHTeamPermissionChecker) Evaluate(snap *scanner.RepoSnapshot) Result {
	tp, ok := snap.TeamPermissions[c.Team]
	if !ok {
		return Result{
			Status:  Fail,
			Message: fmt.Sprintf("team %q not found in repo permissions", c.Team),
		}
	}

	required := permissionLevel(c.Permission)
	actual := permissionLevel(tp.Permission)

	if actual >= required {
		return Result{
			Status:  Pass,
			Message: fmt.Sprintf("team %q has permission %q (>= required %q)", c.Team, tp.Permission, c.Permission),
		}
	}

	return Result{
		Status:  Fail,
		Message: fmt.Sprintf("team %q has permission %q, required at least %q", c.Team, tp.Permission, c.Permission),
	}
}

// GHRepoSettingChecker checks a repo setting field by name.
type GHRepoSettingChecker struct {
	Field string
	Value string
}

func (c *GHRepoSettingChecker) Evaluate(snap *scanner.RepoSnapshot) Result {
	settings := snap.Settings
	var actual string

	switch c.Field {
	case "allow_squash_merge":
		actual = strconv.FormatBool(settings.AllowSquashMerge)
	case "allow_merge_commit":
		actual = strconv.FormatBool(settings.AllowMergeCommit)
	case "allow_rebase_merge":
		actual = strconv.FormatBool(settings.AllowRebaseMerge)
	case "default_branch":
		actual = settings.DefaultBranch
	case "private":
		actual = strconv.FormatBool(settings.Private)
	default:
		return Result{
			Status:  Error,
			Message: fmt.Sprintf("unknown repo setting field: %q", c.Field),
		}
	}

	if actual == c.Value {
		return Result{
			Status:  Pass,
			Message: fmt.Sprintf("repo setting %q = %q as expected", c.Field, c.Value),
		}
	}

	return Result{
		Status:   Fail,
		Message:  fmt.Sprintf("repo setting %q = %q, expected %q", c.Field, actual, c.Value),
		Evidence: []string{fmt.Sprintf("%s=%s (expected %s)", c.Field, actual, c.Value)},
	}
}
