package reporter

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

type TableReporter struct {
	w io.Writer
}

func (r *TableReporter) Report(report *engine.ScanReport) error {
	if err := writeResultRows(r.w, report); err != nil {
		return err
	}

	return writeAggregateReport(r.w, report)
}

func writeResultRows(w io.Writer, report *engine.ScanReport) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	fmt.Fprintln(tw, "REPO\tRULE\tSEVERITY\tSTATUS\tMESSAGE")
	fmt.Fprintln(tw, "----\t----\t--------\t------\t-------")
	for _, result := range report.Results {
		ruleName, severity := rulePresentation(result.Rule)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", result.Repo, ruleName, severity, string(result.Status), result.Message)
	}

	if err := tw.Flush(); err != nil {
		return err
	}
	return nil
}

func rulePresentation(r *rule.Rule) (string, string) {
	if r == nil {
		return "", ""
	}
	return r.Name, string(r.Severity)
}

func writeAggregateReport(w io.Writer, report *engine.ScanReport) error {
	// ── Aggregate summary ──────────────────────────────────────────────────────

	fmt.Fprintln(w)
	fmt.Fprintf(w, "=== Aggregate Report ===\n\n")

	// Per-repo breakdown
	repoStats := report.RepoSummary()
	repos := sortedReposByFailures(repoStats)

	if len(repos) > 0 {
		fmt.Fprintln(w, "Per-repo breakdown:")
		rtw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(rtw, "  REPO\tPASS\tFAIL\tSKIP\tERROR")
		fmt.Fprintln(rtw, "  ----\t----\t----\t----\t-----")
		for _, repo := range repos {
			s := repoStats[repo]
			fmt.Fprintf(rtw, "  %s\t%d\t%d\t%d\t%d\n", repo, s.Pass, s.Fail, s.Skip, s.Error)
		}
		_ = rtw.Flush()
		fmt.Fprintln(w)
	}

	// Top failing rules
	ruleCounts := report.RuleFailureCounts()
	if len(ruleCounts) > 0 {
		rc := sortedRuleCounts(ruleCounts)
		limit := minInt(10, len(rc))

		fmt.Fprintln(w, "Top failing rules (by repo count):")
		ftw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(ftw, "  RULE\tFAILURES")
		fmt.Fprintln(ftw, "  ----\t--------")
		for _, entry := range rc[:limit] {
			fmt.Fprintf(ftw, "  %s\t%d\n", entry.id, entry.count)
		}
		_ = ftw.Flush()
		fmt.Fprintln(w)
	}

	// Severity breakdown
	severityBreakdown := report.SeverityBreakdown()
	if len(severityBreakdown) > 0 {
		severities := []string{"error", "warning", "info"}
		fmt.Fprintln(w, "Severity breakdown (failures only):")
		for _, sev := range severities {
			if count, ok := severityBreakdown[sev]; ok {
				fmt.Fprintf(w, "  %-10s %d\n", sev, count)
			}
		}
		fmt.Fprintln(w)
	}

	// Org-wide totals
	fmt.Fprintf(w, "Org totals: %d repos scanned, %d passed, %d failed\n",
		report.RepoCount(),
		report.PassCount(),
		report.FailureCount(),
	)

	return nil
}

func sortedReposByFailures(repoStats map[string]*engine.RepoStats) []string {
	repos := make([]string, 0, len(repoStats))
	for repo := range repoStats {
		repos = append(repos, repo)
	}
	sort.Slice(repos, func(i, j int) bool {
		fi := repoStats[repos[i]].Fail
		fj := repoStats[repos[j]].Fail
		if fi != fj {
			return fi > fj
		}
		return repos[i] < repos[j]
	})
	return repos
}
