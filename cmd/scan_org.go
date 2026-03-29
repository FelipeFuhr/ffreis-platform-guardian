package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/ffreis/platform-guardian/internal/ai"
	"github.com/ffreis/platform-guardian/internal/baseline"
	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/org"
	"github.com/ffreis/platform-guardian/internal/reporter"
	"github.com/ffreis/platform-guardian/internal/rule"

	"go.uber.org/zap"
)

var scanOrgCmd = &cobra.Command{
	Use:   "scan-org",
	Short: "Scan all repositories in an organization",
	RunE:  runScanOrg,
}

var (
	scanOrgOrg             string
	scanOrgRules           []string
	scanOrgToken           string
	scanOrgFormat          string
	scanOrgFailOn          string
	scanOrgConcurrency     int
	scanOrgBaseline        string
	scanOrgWriteBaseline   string
	scanOrgIncludeArchived bool
	scanOrgIncludeForks    bool
	scanOrgTopics          []string
	scanOrgNamePattern     string
	scanOrgOutputFile      string
	scanOrgAISuggest       bool
	scanOrgAIAPIKey        string
)

func init() {
	scanOrgCmd.Flags().StringVar(&scanOrgOrg, "org", "", "GitHub organization name (required)")
	scanOrgCmd.Flags().StringSliceVar(&scanOrgRules, "rules", nil, "Rule directories or files (required)")
	scanOrgCmd.Flags().StringVar(&scanOrgToken, "token", "", "GitHub token (falls back to GITHUB_TOKEN env)")
	scanOrgCmd.Flags().StringVar(&scanOrgFormat, "format", "table", "Output format: table|summary|json")
	scanOrgCmd.Flags().StringVar(&scanOrgFailOn, "fail-on", "error", "Severity threshold for non-zero exit")
	scanOrgCmd.Flags().IntVar(&scanOrgConcurrency, "concurrency", 10, "Number of concurrent repo scans")
	scanOrgCmd.Flags().StringVar(&scanOrgBaseline, "baseline", "", "Path to baseline file for diff")
	scanOrgCmd.Flags().StringVar(&scanOrgWriteBaseline, "write-baseline", "", "Path to write updated baseline")
	scanOrgCmd.Flags().BoolVar(&scanOrgIncludeArchived, "include-archived", false, "Include archived repos")
	scanOrgCmd.Flags().BoolVar(&scanOrgIncludeForks, "include-forks", false, "Include forked repos")
	scanOrgCmd.Flags().StringSliceVar(&scanOrgTopics, "topic", nil, "Filter repos by topic (repeatable)")
	scanOrgCmd.Flags().StringVar(&scanOrgNamePattern, "name-pattern", "", "Filter repos by name glob pattern")
	scanOrgCmd.Flags().StringVar(&scanOrgOutputFile, "output-file", "", "Also write JSON report to this file path")
	scanOrgCmd.Flags().BoolVar(&scanOrgAISuggest, "ai-suggest", false, "Run AI pattern analysis and fix suggestions")
	scanOrgCmd.Flags().StringVar(&scanOrgAIAPIKey, "ai-api-key", "", "Anthropic API key for AI suggestions (falls back to ANTHROPIC_API_KEY)")

	_ = scanOrgCmd.MarkFlagRequired("org")
	_ = scanOrgCmd.MarkFlagRequired("rules")
}

func runScanOrg(cmd *cobra.Command, args []string) error {
	log := getLogger(cmd)

	token := resolveGitHubToken(scanOrgToken)

	registry, err := loadAndValidateRegistry(scanOrgRules, os.Stderr)
	if err != nil {
		return err
	}

	repos, err := discoverRepos(cmd, token)
	if err != nil {
		return err
	}

	report, failOn, err := scanRepos(cmd, registry, log, repos, token)
	if err != nil {
		return err
	}

	setReportMetadata(report, time.Now())

	report, err = handleBaseline(report)
	if err != nil {
		return err
	}

	if err := maybeWriteJSONReport(report); err != nil {
		return err
	}

	if err := reportToStdout(report); err != nil {
		return err
	}

	maybeRunAIAnalysis(cmd, report)

	exitIfFailures(report, failOn)
	return nil
}

func resolveGitHubToken(flagToken string) string {
	if flagToken != "" {
		return flagToken
	}
	return os.Getenv("GITHUB_TOKEN")
}

func loadAndValidateRegistry(paths []string, stderr *os.File) (*rule.Registry, error) {
	registry, err := rule.Load(paths)
	if err != nil {
		return nil, fmt.Errorf("loading rules: %w", err)
	}

	if errs := rule.Validate(registry); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(stderr, "validation error: %v\n", e)
		}
		return nil, fmt.Errorf("rule validation failed")
	}

	return registry, nil
}

func discoverRepos(cmd *cobra.Command, token string) ([]org.RepoInfo, error) {
	repos, err := org.Discover(cmd.Context(), org.DiscoveryOptions{
		Token:           token,
		Org:             scanOrgOrg,
		IncludeArchived: scanOrgIncludeArchived,
		IncludeForks:    scanOrgIncludeForks,
		Topics:          scanOrgTopics,
		NamePattern:     scanOrgNamePattern,
	})
	if err != nil {
		return nil, fmt.Errorf("discovering repos: %w", err)
	}

	fmt.Fprintf(os.Stderr, "discovered %d repos\n", len(repos))
	return repos, nil
}

func scanRepos(
	cmd *cobra.Command,
	registry *rule.Registry,
	log *zap.Logger,
	repos []org.RepoInfo,
	token string,
) (*engine.ScanReport, rule.Severity, error) {
	eng := engine.NewEngine(registry, log)
	failOn := rule.Severity(scanOrgFailOn)

	pool := org.NewWorkerPool(scanOrgConcurrency, eng, log)
	report, err := pool.ScanAll(cmd.Context(), repos, token, failOn)
	if err != nil {
		return nil, "", fmt.Errorf("scanning repos: %w", err)
	}
	return report, failOn, nil
}

func setReportMetadata(report *engine.ScanReport, now time.Time) {
	report.RunID = fmt.Sprintf("run-%d", now.UnixNano())
	report.GeneratedAt = now.UTC().Format("2006-01-02T15:04:05Z07:00")
}

func handleBaseline(report *engine.ScanReport) (*engine.ScanReport, error) {
	if scanOrgBaseline == "" {
		return report, nil
	}

	b, err := baseline.Load(scanOrgBaseline)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading baseline: %w", err)
	}
	if b != nil {
		report = baseline.FilterNew(report, b)
	}

	if scanOrgWriteBaseline == "" {
		return report, nil
	}

	updated := baseline.Update(baselineOrEmpty(b), report)
	if err := baseline.Save(scanOrgWriteBaseline, updated); err != nil {
		return nil, fmt.Errorf("saving baseline: %w", err)
	}

	return report, nil
}

func baselineOrEmpty(b *baseline.Baseline) *baseline.Baseline {
	if b != nil {
		return b
	}
	return &baseline.Baseline{}
}

func maybeWriteJSONReport(report *engine.ScanReport) error {
	if scanOrgOutputFile == "" {
		return nil
	}
	if err := writeJSONReport(scanOrgOutputFile, report); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}
	return nil
}

func reportToStdout(report *engine.ScanReport) error {
	rep, err := reporter.New(scanOrgFormat, os.Stdout)
	if err != nil {
		return fmt.Errorf("creating reporter: %w", err)
	}

	if err := rep.Report(report); err != nil {
		return fmt.Errorf("reporting: %w", err)
	}
	return nil
}

func maybeRunAIAnalysis(cmd *cobra.Command, report *engine.ScanReport) {
	if !scanOrgAISuggest {
		return
	}
	if err := runAIAnalysis(cmd, report); err != nil {
		// Non-fatal: print warning and continue.
		fmt.Fprintf(os.Stderr, "warning: AI analysis failed: %v\n", err)
	}
}

func exitIfFailures(report *engine.ScanReport, failOn rule.Severity) {
	if report.HasFailures(failOn) {
		os.Exit(1)
	}
}

func writeJSONReport(path string, report *engine.ScanReport) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	rep, err := reporter.New("json", f)
	if err != nil {
		return err
	}
	return rep.Report(report)
}

func runAIAnalysis(cmd *cobra.Command, report *engine.ScanReport) error {
	analysis := ai.Analyze(report, nil)

	// Optionally enhance suggestions via Claude
	apiKey := scanOrgAIAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey != "" {
		enhanced, err := ai.EnhanceSuggestions(cmd.Context(), analysis.Suggestions, apiKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: Claude enhancement failed: %v\n", err)
		} else {
			analysis.Suggestions = enhanced
		}
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprint(os.Stdout, ai.FormatAnalysis(analysis))
	return nil
}
