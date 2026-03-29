package check

import (
	"testing"

	"github.com/ffreis/platform-guardian/internal/scanner"
)

func mockSnapshot() *scanner.RepoSnapshot {
	snap := scanner.NewSnapshot("test/repo")
	snap.FilePaths = []string{
		"README.md",
		"Makefile",
		".github/workflows/ci.yml",
		"src/main.go",
	}
	snap.FileContents = map[string]string{
		"README.md": "# My Project\nThis is a test project.\nVersion: 1.0.0",
		"Makefile":  "build:\n\tgo build ./...",
	}
	return snap
}

func TestFileExistsPass(t *testing.T) {
	snap := mockSnapshot()
	checker := &FileExistsChecker{Path: "README.md"}
	result := checker.Evaluate(snap)
	if result.Status != Pass {
		t.Errorf("expected Pass, got %s: %s", result.Status, result.Message)
	}
}

func TestFileExistsFail(t *testing.T) {
	snap := mockSnapshot()
	checker := &FileExistsChecker{Path: "CODEOWNERS"}
	result := checker.Evaluate(snap)
	if result.Status != Fail {
		t.Errorf("expected Fail, got %s: %s", result.Status, result.Message)
	}
}

func TestFileExistsGlob(t *testing.T) {
	snap := mockSnapshot()
	checker := &FileExistsChecker{Path: ".github/workflows/*.yml"}
	result := checker.Evaluate(snap)
	if result.Status != Pass {
		t.Errorf("expected Pass for glob match, got %s: %s", result.Status, result.Message)
	}
}

func TestFileContainsPass(t *testing.T) {
	snap := mockSnapshot()
	checker := &FileContainsChecker{Path: "README.md", Pattern: "Version: [0-9]+"}
	result := checker.Evaluate(snap)
	if result.Status != Pass {
		t.Errorf("expected Pass, got %s: %s", result.Status, result.Message)
	}
}

func TestFileContainsFail(t *testing.T) {
	snap := mockSnapshot()
	checker := &FileContainsChecker{Path: "README.md", Pattern: "NONEXISTENT_PATTERN_XYZ"}
	result := checker.Evaluate(snap)
	if result.Status != Fail {
		t.Errorf("expected Fail, got %s: %s", result.Status, result.Message)
	}
}
