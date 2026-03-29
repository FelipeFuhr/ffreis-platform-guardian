package check

import "testing"

func TestFileAbsentPass(t *testing.T) {
	snap := mockSnapshot()
	checker := &FileAbsentChecker{Path: "CODEOWNERS"}
	result := checker.Evaluate(snap)
	if result.Status != Pass {
		t.Fatalf("expected Pass, got %s: %s", result.Status, result.Message)
	}
}

func TestFileAbsentFail(t *testing.T) {
	snap := mockSnapshot()
	checker := &FileAbsentChecker{Path: "Makefile"}
	result := checker.Evaluate(snap)
	if result.Status != Fail {
		t.Fatalf("expected Fail, got %s: %s", result.Status, result.Message)
	}
}

func TestFileNotContainsPass(t *testing.T) {
	snap := mockSnapshot()
	checker := &FileNotContainsChecker{Path: readmePath, Pattern: "DROP TABLE"}
	result := checker.Evaluate(snap)
	if result.Status != Pass {
		t.Fatalf("expected Pass, got %s: %s", result.Status, result.Message)
	}
}

func TestFileNotContainsFail(t *testing.T) {
	snap := mockSnapshot()
	checker := &FileNotContainsChecker{Path: readmePath, Pattern: "Version: [0-9]+"}
	result := checker.Evaluate(snap)
	if result.Status != Fail {
		t.Fatalf("expected Fail, got %s: %s", result.Status, result.Message)
	}
}
