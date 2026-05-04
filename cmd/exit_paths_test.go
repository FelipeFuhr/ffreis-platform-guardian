package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
	"github.com/spf13/cobra"
)

func TestExitIfFailures_ExitsWithCode1(t *testing.T) {
	rep := &engine.ScanReport{
		Results: []engine.RuleResult{
			{Repo: "org/repo", Rule: &rule.Rule{ID: "r1", Severity: rule.SeverityError}, Status: engine.StatusFail},
		},
	}

	err := exitIfFailures(rep, rule.SeverityError)
	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.Code != exitError {
		t.Fatalf("ExitError.Code = %d, want %d", exitErr.Code, exitError)
	}
}

func TestExecute_ExitsOnCommandError(t *testing.T) {
	rootCmd.SetArgs([]string{"definitely-not-a-command"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if code := Execute(); code != exitError {
		t.Fatalf("Execute() code = %d, want %d", code, exitError)
	}
}

func TestExecuteCommand_WritesErrorText(t *testing.T) {
	t.Parallel()

	command := &cobra.Command{
		RunE: func(*cobra.Command, []string) error {
			return &ExitError{Code: 7, Err: errors.New("boom")}
		},
	}

	var stderr bytes.Buffer
	code := executeCommand(command, &stderr)
	if code != 7 {
		t.Fatalf("executeCommand() code = %d, want 7", code)
	}
	if got := stderr.String(); got != "error: boom\n" {
		t.Fatalf("executeCommand() stderr = %q", got)
	}
}
