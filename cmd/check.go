package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/reporter"
	"github.com/ffreis/platform-guardian/internal/rule"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check a single repository against rules",
	RunE:  runCheck,
}

var (
	checkRepo   string
	checkRules  []string
	checkToken  string
	checkRef    string
	checkFormat string
	checkFailOn string
)

func init() {
	checkCmd.Flags().StringVar(&checkRepo, "repo", "", "Repository in org/repo format (required)")
	checkCmd.Flags().StringSliceVar(&checkRules, "rules", nil, "Rule directories or files (required)")
	checkCmd.Flags().StringVar(&checkToken, "token", "", "GitHub token (falls back to GITHUB_TOKEN env)")
	checkCmd.Flags().StringVar(&checkRef, "ref", "", "Git ref to scan (commit SHA, branch, or tag). Defaults to repository default branch.")
	checkCmd.Flags().StringVar(&checkFormat, "format", "table", "Output format: table|json|sarif|annotations")
	checkCmd.Flags().StringVar(&checkFailOn, "fail-on", "error", "Severity threshold for non-zero exit: error|warning|info")

	_ = checkCmd.MarkFlagRequired("repo")
	_ = checkCmd.MarkFlagRequired("rules")
}

func runCheck(cmd *cobra.Command, args []string) error {
	log := getLogger(cmd)

	// Token fallback
	token := checkToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	// Load rules
	registry, err := rule.Load(checkRules)
	if err != nil {
		return fmt.Errorf("loading rules: %w", err)
	}

	// Validate
	if errs := rule.Validate(registry); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "validation error: %v\n", e)
		}
		return fmt.Errorf("rule validation failed")
	}

	// Run check
	eng := engine.NewEngine(registry, log)
	failOn := rule.Severity(checkFailOn)

	report, err := eng.Check(cmd.Context(), engine.ScanOptions{
		Token:  token,
		Repo:   checkRepo,
		Ref:    checkRef,
		FailOn: failOn,
		Format: checkFormat,
	})
	if err != nil {
		return fmt.Errorf("running check: %w", err)
	}

	// Report
	rep, err := reporter.New(checkFormat, os.Stdout)
	if err != nil {
		return fmt.Errorf("creating reporter: %w", err)
	}

	if err := rep.Report(report); err != nil {
		return fmt.Errorf("reporting: %w", err)
	}

	// Exit 1 if failures above threshold
	if report.HasFailures(failOn) {
		os.Exit(1)
	}

	return nil
}
