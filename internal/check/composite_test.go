package check

import (
	"testing"

	"github.com/ffreis/platform-guardian/internal/scanner"
)

type alwaysPass struct{}

func (a *alwaysPass) Evaluate(_ *scanner.RepoSnapshot) Result {
	return Result{Status: Pass, Message: "always passes"}
}

type alwaysFail struct{}

func (a *alwaysFail) Evaluate(_ *scanner.RepoSnapshot) Result {
	return Result{Status: Fail, Message: "always fails"}
}

func compositeSnap() *scanner.RepoSnapshot {
	return scanner.NewSnapshot("test/repo")
}

func TestCompositeAND_AllPass(t *testing.T) {
	checker := &CompositeChecker{
		Operator: "AND",
		Checks:   []Checker{&alwaysPass{}, &alwaysPass{}, &alwaysPass{}},
	}
	result := checker.Evaluate(compositeSnap())
	if result.Status != Pass {
		t.Errorf("expected Pass, got %s: %s", result.Status, result.Message)
	}
}

func TestCompositeAND_OneFails(t *testing.T) {
	checker := &CompositeChecker{
		Operator: "AND",
		Checks:   []Checker{&alwaysPass{}, &alwaysFail{}, &alwaysPass{}},
	}
	result := checker.Evaluate(compositeSnap())
	if result.Status != Fail {
		t.Errorf("expected Fail (short-circuit), got %s: %s", result.Status, result.Message)
	}
}

func TestCompositeOR_OnePasses(t *testing.T) {
	checker := &CompositeChecker{
		Operator: "OR",
		Checks:   []Checker{&alwaysFail{}, &alwaysPass{}, &alwaysFail{}},
	}
	result := checker.Evaluate(compositeSnap())
	if result.Status != Pass {
		t.Errorf("expected Pass (one passes), got %s: %s", result.Status, result.Message)
	}
}

func TestCompositeOR_AllFail(t *testing.T) {
	checker := &CompositeChecker{
		Operator: "OR",
		Checks:   []Checker{&alwaysFail{}, &alwaysFail{}},
	}
	result := checker.Evaluate(compositeSnap())
	if result.Status != Fail {
		t.Errorf("expected Fail, got %s: %s", result.Status, result.Message)
	}
}

func TestCompositeNOT_Inverts(t *testing.T) {
	// NOT(fail) → pass
	checkerPass := &CompositeChecker{
		Operator: "NOT",
		Checks:   []Checker{&alwaysFail{}},
	}
	result := checkerPass.Evaluate(compositeSnap())
	if result.Status != Pass {
		t.Errorf("expected Pass (NOT fail), got %s: %s", result.Status, result.Message)
	}

	// NOT(pass) → fail
	checkerFail := &CompositeChecker{
		Operator: "NOT",
		Checks:   []Checker{&alwaysPass{}},
	}
	result2 := checkerFail.Evaluate(compositeSnap())
	if result2.Status != Fail {
		t.Errorf("expected Fail (NOT pass), got %s: %s", result2.Status, result2.Message)
	}
}
