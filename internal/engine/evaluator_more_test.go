package engine

import (
	"testing"

	"github.com/ffreis/platform-guardian/internal/check"
	"github.com/ffreis/platform-guardian/internal/rule"
)

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

			switch tc.want.(type) {
			case *check.FileExistsChecker:
				if _, ok := got.(*check.FileExistsChecker); !ok {
					t.Fatalf("expected FileExistsChecker, got %T", got)
				}
			case *check.FileAbsentChecker:
				if _, ok := got.(*check.FileAbsentChecker); !ok {
					t.Fatalf("expected FileAbsentChecker, got %T", got)
				}
			case *check.FileContainsChecker:
				if _, ok := got.(*check.FileContainsChecker); !ok {
					t.Fatalf("expected FileContainsChecker, got %T", got)
				}
			case *check.FileNotContainsChecker:
				if _, ok := got.(*check.FileNotContainsChecker); !ok {
					t.Fatalf("expected FileNotContainsChecker, got %T", got)
				}
			case *check.TFProviderReqChecker:
				if _, ok := got.(*check.TFProviderReqChecker); !ok {
					t.Fatalf("expected TFProviderReqChecker, got %T", got)
				}
			case *check.TFBackendConfigChecker:
				if _, ok := got.(*check.TFBackendConfigChecker); !ok {
					t.Fatalf("expected TFBackendConfigChecker, got %T", got)
				}
			case *check.TFRequiredTagsChecker:
				if _, ok := got.(*check.TFRequiredTagsChecker); !ok {
					t.Fatalf("expected TFRequiredTagsChecker, got %T", got)
				}
			case *check.TFResourceForbidChecker:
				if _, ok := got.(*check.TFResourceForbidChecker); !ok {
					t.Fatalf("expected TFResourceForbidChecker, got %T", got)
				}
			case *check.TFModuleUsedChecker:
				if _, ok := got.(*check.TFModuleUsedChecker); !ok {
					t.Fatalf("expected TFModuleUsedChecker, got %T", got)
				}
			case *check.TFVariableReqChecker:
				if _, ok := got.(*check.TFVariableReqChecker); !ok {
					t.Fatalf("expected TFVariableReqChecker, got %T", got)
				}
			case *check.GHBranchProtectChecker:
				if _, ok := got.(*check.GHBranchProtectChecker); !ok {
					t.Fatalf("expected GHBranchProtectChecker, got %T", got)
				}
			case *check.GHTeamPermissionChecker:
				if _, ok := got.(*check.GHTeamPermissionChecker); !ok {
					t.Fatalf("expected GHTeamPermissionChecker, got %T", got)
				}
			case *check.GHRepoSettingChecker:
				if _, ok := got.(*check.GHRepoSettingChecker); !ok {
					t.Fatalf("expected GHRepoSettingChecker, got %T", got)
				}
			case *check.CompositeChecker:
				cc, ok := got.(*check.CompositeChecker)
				if !ok {
					t.Fatalf("expected CompositeChecker, got %T", got)
				}
				if cc.Operator != "AND" || len(cc.Checks) != 2 {
					t.Fatalf("unexpected composite checker: operator=%q checks=%d", cc.Operator, len(cc.Checks))
				}
			default:
				t.Fatalf("unknown want type")
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
