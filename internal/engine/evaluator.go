package engine

import (
	"fmt"

	"github.com/ffreis/platform-guardian/internal/check"
	"github.com/ffreis/platform-guardian/internal/rule"
	"github.com/ffreis/platform-guardian/internal/scanner"
)

// Evaluate dispatches to the appropriate checker based on the rule type.
func Evaluate(r *rule.Rule, snap *scanner.RepoSnapshot) RuleResult {
	checker, err := buildChecker(r.Check)
	if err != nil {
		return RuleResult{
			Repo:    snap.Repo,
			Rule:    r,
			Status:  StatusError,
			Message: fmt.Sprintf("failed to build checker: %v", err),
		}
	}

	checkResult := checker.Evaluate(snap)

	return RuleResult{
		Repo:        snap.Repo,
		Rule:        r,
		Status:      CheckStatus(checkResult.Status),
		Message:     checkResult.Message,
		Evidence:    checkResult.Evidence,
		Remediation: r.Remediation.Description,
	}
}

func buildChecker(spec rule.CheckSpec) (check.Checker, error) {
	switch {
	case spec.FileExists != nil:
		return &check.FileExistsChecker{Path: spec.FileExists.Path}, nil

	case spec.FileAbsent != nil:
		return &check.FileAbsentChecker{Path: spec.FileAbsent.Path}, nil

	case spec.FileContains != nil:
		return &check.FileContainsChecker{
			Path:    spec.FileContains.Path,
			Pattern: spec.FileContains.Pattern,
		}, nil

	case spec.FileNotContains != nil:
		return &check.FileNotContainsChecker{
			Path:    spec.FileNotContains.Path,
			Pattern: spec.FileNotContains.Pattern,
		}, nil

	case spec.TFProviderReq != nil:
		return &check.TFProviderReqChecker{
			Source:  spec.TFProviderReq.Source,
			Version: spec.TFProviderReq.Version,
		}, nil

	case spec.TFBackendConfig != nil:
		return &check.TFBackendConfigChecker{
			Type:   spec.TFBackendConfig.Type,
			Fields: spec.TFBackendConfig.Fields,
		}, nil

	case spec.TFRequiredTags != nil:
		return &check.TFRequiredTagsChecker{
			Tags: spec.TFRequiredTags.Tags,
		}, nil

	case spec.TFResourceForbid != nil:
		return &check.TFResourceForbidChecker{
			Type: spec.TFResourceForbid.Type,
		}, nil

	case spec.TFModuleUsed != nil:
		return &check.TFModuleUsedChecker{
			Source: spec.TFModuleUsed.Source,
		}, nil

	case spec.TFVariableReq != nil:
		return &check.TFVariableReqChecker{
			Name: spec.TFVariableReq.Name,
			Type: spec.TFVariableReq.Type,
		}, nil

	case spec.GHBranchProtect != nil:
		return &check.GHBranchProtectChecker{
			Branch:              spec.GHBranchProtect.Branch,
			RequirePRReviews:    spec.GHBranchProtect.RequirePRReviews,
			RequireStatusChecks: spec.GHBranchProtect.RequireStatusChecks,
		}, nil

	case spec.GHTeamPermission != nil:
		return &check.GHTeamPermissionChecker{
			Team:       spec.GHTeamPermission.Team,
			Permission: spec.GHTeamPermission.Permission,
		}, nil

	case spec.GHRepoSetting != nil:
		return &check.GHRepoSettingChecker{
			Field: spec.GHRepoSetting.Field,
			Value: spec.GHRepoSetting.Value,
		}, nil

	case spec.Composite != nil:
		var children []check.Checker
		for _, childSpec := range spec.Composite.Checks {
			child, err := buildChecker(childSpec)
			if err != nil {
				return nil, fmt.Errorf("building composite child checker: %w", err)
			}
			children = append(children, child)
		}
		return &check.CompositeChecker{
			Operator: spec.Composite.Operator,
			Checks:   children,
		}, nil
	}

	return nil, fmt.Errorf("no check spec defined")
}
