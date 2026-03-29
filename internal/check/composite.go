package check

import (
	"fmt"

	"github.com/ffreis/platform-guardian/internal/scanner"
)

// CompositeChecker implements AND/OR/NOT logic over child checkers.
type CompositeChecker struct {
	Operator string
	Checks   []Checker
}

func (c *CompositeChecker) Evaluate(snap *scanner.RepoSnapshot) Result {
	switch c.Operator {
	case "AND":
		return c.evaluateAND(snap)
	case "OR":
		return c.evaluateOR(snap)
	case "NOT":
		return c.evaluateNOT(snap)
	default:
		return Result{
			Status:  Error,
			Message: fmt.Sprintf("unknown composite operator: %q", c.Operator),
		}
	}
}

func (c *CompositeChecker) evaluateAND(snap *scanner.RepoSnapshot) Result {
	var evidence []string
	for _, checker := range c.Checks {
		result := checker.Evaluate(snap)
		if result.Status != Pass {
			// Short-circuit
			return Result{
				Status:   Fail,
				Message:  fmt.Sprintf("AND: child check failed: %s", result.Message),
				Evidence: result.Evidence,
			}
		}
		evidence = append(evidence, result.Message)
	}
	return Result{
		Status:   Pass,
		Message:  "AND: all checks passed",
		Evidence: evidence,
	}
}

func (c *CompositeChecker) evaluateOR(snap *scanner.RepoSnapshot) Result {
	var failures []string
	for _, checker := range c.Checks {
		result := checker.Evaluate(snap)
		if result.Status == Pass {
			// Short-circuit
			return Result{
				Status:  Pass,
				Message: fmt.Sprintf("OR: at least one check passed: %s", result.Message),
			}
		}
		failures = append(failures, result.Message)
	}
	return Result{
		Status:   Fail,
		Message:  "OR: all checks failed",
		Evidence: failures,
	}
}

func (c *CompositeChecker) evaluateNOT(snap *scanner.RepoSnapshot) Result {
	if len(c.Checks) == 0 {
		return Result{
			Status:  Error,
			Message: "NOT: no child checks",
		}
	}
	result := c.Checks[0].Evaluate(snap)
	if result.Status == Pass {
		return Result{
			Status:  Fail,
			Message: fmt.Sprintf("NOT: check passed but should have failed: %s", result.Message),
		}
	}
	return Result{
		Status:  Pass,
		Message: fmt.Sprintf("NOT: check correctly failed: %s", result.Message),
	}
}
