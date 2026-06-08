package engine

import (
	"reflect"
	"testing"

	"github.com/ffreis/platform-guardian/internal/check"
	"github.com/ffreis/platform-guardian/internal/rule"
)

// assertCheckerType verifies that got has the same concrete type as want,
// and returns the concrete value cast to *check.CompositeChecker when
// applicable (nil otherwise).
func assertCheckerType(t *testing.T, got check.Checker, want any) *check.CompositeChecker {
	t.Helper()
	gotType := reflect.TypeOf(got)
	wantType := reflect.TypeOf(want)
	if gotType != wantType {
		t.Fatalf("expected %s, got %T", wantType, got)
	}
	if cc, ok := got.(*check.CompositeChecker); ok {
		return cc
	}
	return nil
}

func TestBuildChecker_AllSpecTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		spec rule.CheckSpec
		want any
	}{
		{name: "file_exists", spec: rule.CheckSpec{FileExists: &rule.FileExistsCheck{Path: "README.md"}}, want: &check.FileExistsChecker{}},
		{name: "file_absent", spec: rule.CheckSpec{FileAbsent: &rule.FileAbsentCheck{Path: "CODEOWNERS"}}, want: &check.FileAbsentChecker{}},
		{name: "file_contains", spec: rule.CheckSpec{FileContains: &rule.FileContainsCheck{Path: "README.md", Pattern: "x"}}, want: &check.FileContainsChecker{}},
		{name: "file_not_contains", spec: rule.CheckSpec{FileNotContains: &rule.FileNotContainsCheck{Path: "README.md", Pattern: "x"}}, want: &check.FileNotContainsChecker{}},
		{name: "tf_provider_required", spec: rule.CheckSpec{TFProviderReq: &rule.TFProviderReqCheck{Source: "hashicorp/aws", Version: ">= 1.0"}}, want: &check.TFProviderReqChecker{}},
		{name: "tf_backend_config", spec: rule.CheckSpec{TFBackendConfig: &rule.TFBackendConfigCheck{Type: "s3", Fields: map[string]string{"bucket": "x"}}}, want: &check.TFBackendConfigChecker{}},
		{name: "tf_required_tags", spec: rule.CheckSpec{TFRequiredTags: &rule.TFRequiredTagsCheck{Tags: []string{"env"}}}, want: &check.TFRequiredTagsChecker{}},
		{name: "tf_resource_forbidden", spec: rule.CheckSpec{TFResourceForbid: &rule.TFResourceForbidCheck{Type: "aws_iam_user"}}, want: &check.TFResourceForbidChecker{}},
		{name: "tf_module_used", spec: rule.CheckSpec{TFModuleUsed: &rule.TFModuleUsedCheck{Source: "terraform-modules"}}, want: &check.TFModuleUsedChecker{}},
		{name: "tf_variable_required", spec: rule.CheckSpec{TFVariableReq: &rule.TFVariableReqCheck{Name: "region", Type: "string"}}, want: &check.TFVariableReqChecker{}},
		{name: "gh_branch_protection", spec: rule.CheckSpec{GHBranchProtect: &rule.GHBranchProtectCheck{Branch: "main", RequirePRReviews: true}}, want: &check.GHBranchProtectChecker{}},
		{name: "gh_team_permission", spec: rule.CheckSpec{GHTeamPermission: &rule.GHTeamPermissionCheck{Team: "platform", Permission: "write"}}, want: &check.GHTeamPermissionChecker{}},
		{name: "gh_repo_setting", spec: rule.CheckSpec{GHRepoSetting: &rule.GHRepoSettingCheck{Field: "default_branch", Value: "main"}}, want: &check.GHRepoSettingChecker{}},
		{
			name: "composite",
			spec: rule.CheckSpec{Composite: &rule.CompositeCheck{
				Operator: "AND",
				Checks: []rule.CheckSpec{
					{FileExists: &rule.FileExistsCheck{Path: "README.md"}},
					{GHRepoSetting: &rule.GHRepoSettingCheck{Field: "default_branch", Value: "main"}},
				},
			}},
			want: &check.CompositeChecker{},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := buildChecker(tc.spec)
			if err != nil {
				t.Fatalf("buildChecker() error = %v", err)
			}

			cc := assertCheckerType(t, got, tc.want)
			if cc != nil && (cc.Operator != "AND" || len(cc.Checks) != 2) {
				t.Fatalf("unexpected composite checker: operator=%q checks=%d", cc.Operator, len(cc.Checks))
			}
		})
	}
}

func TestBuildChecker_NoSpecErrors(t *testing.T) {
	t.Parallel()

	if _, err := buildChecker(rule.CheckSpec{}); err == nil {
		t.Fatalf("expected error")
	}
}
