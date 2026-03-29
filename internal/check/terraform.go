package check

import (
	"fmt"
	"path"
	"strings"

	"github.com/ffreis/platform-guardian/internal/scanner"
)

// TFProviderReqChecker checks that at least one module has the required provider.
type TFProviderReqChecker struct {
	Source  string
	Version string
}

func (c *TFProviderReqChecker) Evaluate(snap *scanner.RepoSnapshot) Result {
	for _, module := range snap.TFModules {
		for _, p := range module.Providers {
			if p.Source == c.Source {
				// If version specified, check it's compatible (simple string contains)
				if c.Version == "" || strings.Contains(p.Version, strings.TrimSpace(c.Version)) {
					return Result{
						Status:   Pass,
						Message:  fmt.Sprintf("provider %s found in %s", c.Source, module.Path),
						Evidence: []string{fmt.Sprintf("provider %s version %s in %s", p.Source, p.Version, module.Path)},
					}
				}
			}
		}
	}

	return Result{
		Status:  Fail,
		Message: fmt.Sprintf("required provider %s not found", c.Source),
	}
}

// TFBackendConfigChecker checks that any module has a backend with matching type and fields.
type TFBackendConfigChecker struct {
	Type   string
	Fields map[string]string
}

func (c *TFBackendConfigChecker) Evaluate(snap *scanner.RepoSnapshot) Result {
	for _, module := range snap.TFModules {
		if module.Backend == nil {
			continue
		}
		if module.Backend.Type != c.Type {
			continue
		}
		// Check all required fields
		allFieldsPresent := true
		var missingFields []string
		for k, v := range c.Fields {
			actual, ok := module.Backend.Config[k]
			if !ok || (v != "" && actual != v) {
				allFieldsPresent = false
				missingFields = append(missingFields, k)
			}
		}
		if allFieldsPresent {
			return Result{
				Status:   Pass,
				Message:  fmt.Sprintf("backend type %s found in %s", c.Type, module.Path),
				Evidence: []string{fmt.Sprintf("backend %s in %s", c.Type, module.Path)},
			}
		}
		return Result{
			Status:  Fail,
			Message: fmt.Sprintf("backend type %s found but missing fields: %v", c.Type, missingFields),
		}
	}

	return Result{
		Status:  Fail,
		Message: fmt.Sprintf("backend type %s not found", c.Type),
	}
}

// TFRequiredTagsChecker checks all resources across all modules have all required tags.
type TFRequiredTagsChecker struct {
	Tags []string
}

func (c *TFRequiredTagsChecker) Evaluate(snap *scanner.RepoSnapshot) Result {
	var failingResources []string

	for _, module := range snap.TFModules {
		for _, res := range module.Resources {
			for _, tag := range c.Tags {
				if _, ok := res.Labels[tag]; !ok {
					failingResources = append(failingResources,
						fmt.Sprintf("%s.%s (missing tag: %s)", res.Type, res.Name, tag))
				}
			}
		}
	}

	if len(failingResources) > 0 {
		return Result{
			Status:   Fail,
			Message:  fmt.Sprintf("%d resources missing required tags", len(failingResources)),
			Evidence: failingResources,
		}
	}

	if len(snap.TFModules) == 0 {
		return Result{
			Status:  Skip,
			Message: "no terraform modules found",
		}
	}

	return Result{
		Status:  Pass,
		Message: "all resources have required tags",
	}
}

// TFResourceForbidChecker checks that NO resource has the forbidden type.
type TFResourceForbidChecker struct {
	Type string
}

func (c *TFResourceForbidChecker) Evaluate(snap *scanner.RepoSnapshot) Result {
	for _, module := range snap.TFModules {
		for _, res := range module.Resources {
			matched, err := path.Match(c.Type, res.Type)
			if err == nil && matched {
				return Result{
					Status:   Fail,
					Message:  fmt.Sprintf("forbidden resource type %s found: %s.%s in %s", c.Type, res.Type, res.Name, module.Path),
					Evidence: []string{fmt.Sprintf("%s.%s in %s", res.Type, res.Name, module.Path)},
				}
			}
		}
	}

	return Result{
		Status:  Pass,
		Message: fmt.Sprintf("no forbidden resource type %s found", c.Type),
	}
}

// TFVariableReqChecker checks that a required variable is declared across the modules.
// If Type is non-empty, the variable's declared type must match.
type TFVariableReqChecker struct {
	Name string
	Type string
}

func (c *TFVariableReqChecker) Evaluate(snap *scanner.RepoSnapshot) Result {
	for _, module := range snap.TFModules {
		for _, v := range module.Variables {
			if v.Name != c.Name {
				continue
			}
			if c.Type != "" && v.Type != c.Type {
				return Result{
					Status:   Fail,
					Message:  fmt.Sprintf("variable %q found in %s but type is %q, expected %q", c.Name, module.Path, v.Type, c.Type),
					Evidence: []string{fmt.Sprintf("variable %q type=%q in %s", v.Name, v.Type, module.Path)},
				}
			}
			return Result{
				Status:   Pass,
				Message:  fmt.Sprintf("required variable %q declared in %s", c.Name, module.Path),
				Evidence: []string{fmt.Sprintf("variable %q in %s", v.Name, module.Path)},
			}
		}
	}
	return Result{
		Status:  Fail,
		Message: fmt.Sprintf("required variable %q not declared in any module", c.Name),
	}
}

// TFModuleUsedChecker checks that at least one module call matches the source glob.
type TFModuleUsedChecker struct {
	Source string
}

func (c *TFModuleUsedChecker) Evaluate(snap *scanner.RepoSnapshot) Result {
	for _, module := range snap.TFModules {
		for _, mc := range module.Modules {
			matched, err := path.Match(c.Source, mc.Source)
			if err == nil && matched {
				return Result{
					Status:   Pass,
					Message:  fmt.Sprintf("module %s used in %s", mc.Source, module.Path),
					Evidence: []string{fmt.Sprintf("module %s in %s", mc.Name, module.Path)},
				}
			}
			// Also check simple contains for non-glob patterns
			if !strings.ContainsAny(c.Source, "*?[") && strings.Contains(mc.Source, c.Source) {
				return Result{
					Status:   Pass,
					Message:  fmt.Sprintf("module %s used in %s", mc.Source, module.Path),
					Evidence: []string{fmt.Sprintf("module %s in %s", mc.Name, module.Path)},
				}
			}
		}
	}

	return Result{
		Status:  Fail,
		Message: fmt.Sprintf("module with source %q not found", c.Source),
	}
}
