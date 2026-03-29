package cmd

import (
	"os"
	"os/exec"
	"testing"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

func TestExitIfFailures_ExitsWithCode1(t *testing.T) {
	if os.Getenv("WANT_EXIT_HELPER") == "1" {
		rep := &engine.ScanReport{
			Results: []engine.RuleResult{
				{Repo: "org/repo", Rule: &rule.Rule{ID: "r1", Severity: rule.SeverityError}, Status: engine.StatusFail},
			},
		}
		exitIfFailures(rep, rule.SeverityError)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestExitIfFailures_ExitsWithCode1")
	cmd.Env = append(os.Environ(), "WANT_EXIT_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit")
	}
	if ee, ok := err.(*exec.ExitError); ok {
		if ee.ExitCode() != 1 {
			t.Fatalf("expected exit code 1, got %d", ee.ExitCode())
		}
		return
	}
	t.Fatalf("expected ExitError, got %T", err)
}

func TestExecute_ExitsOnCommandError(t *testing.T) {
	if os.Getenv("WANT_EXECUTE_HELPER") == "1" {
		rootCmd.SetArgs([]string{"definitely-not-a-command"})
		Execute()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestExecute_ExitsOnCommandError")
	cmd.Env = append(os.Environ(), "WANT_EXECUTE_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit")
	}
	if ee, ok := err.(*exec.ExitError); ok {
		if ee.ExitCode() != 1 {
			t.Fatalf("expected exit code 1, got %d", ee.ExitCode())
		}
		return
	}
	t.Fatalf("expected ExitError, got %T", err)
}
