package reporter

import (
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

	// scan-fix(golangci:errcheck): route through errWriter (see helpers.go) so a
	// real write failure surfaces instead of being silently dropped call-by-call.
	twEw := &errWriter{w: tw}
	twEw.println("REPO\tRULE\tSEVERITY\tSTATUS\tMESSAGE")
	twEw.println("----\t----\t--------\t------\t-------")
	for _, result := range report.Results {
		ruleName, severity := rulePresentation(result.Rule)
		twEw.printf("%s\t%s\t%s\t%s\t%s\n", result.Repo, ruleName, severity, string(result.Status), result.Message)
	}
	if twEw.err != nil {
		return twEw.err
	}

	return tw.Flush()
}

func rulePresentation(r *rule.Rule) (string, string) {
	if r == nil {
		return "", ""
	}
	return r.Name, string(r.Severity)
}

func writeAggregateReport(w io.Writer, report *engine.ScanReport) error {
	// ── Aggregate summary ──────────────────────────────────────────────────────

	// scan-fix(golangci:errcheck): route through errWriter (see helpers.go) so a
	// real write failure surfaces instead of being silently dropped call-by-call.
	// This also fixes two previously-swallowed `_ = rtw.Flush()` / `_ = ftw.Flush()`
	// calls below, which used to drop a genuine flush error on the floor.
	ew := &errWriter{w: w}
	ew.println()
	ew.printf("=== Aggregate Report ===\n\n")

	// Per-repo breakdown
	repoStats := report.RepoSummary()
	repos := sortedReposByFailures(repoStats)

	if len(repos) > 0 {
		ew.println("Per-repo breakdown:")
		rtw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		rtwEw := &errWriter{w: rtw}
		rtwEw.println("  REPO\tPASS\tFAIL\tSKIP\tERROR")
		rtwEw.println("  ----\t----\t----\t----\t-----")
		for _, repo := range repos {
			s := repoStats[repo]
			rtwEw.printf("  %s\t%d\t%d\t%d\t%d\n", repo, s.Pass, s.Fail, s.Skip, s.Error)
		}
		if rtwEw.err != nil {
			return rtwEw.err
		}
		if err := rtw.Flush(); err != nil {
			return err
		}
		ew.println()
	}

	// Top failing rules
	ruleCounts := report.RuleFailureCounts()
	if len(ruleCounts) > 0 {
		rc := sortedRuleCounts(ruleCounts)
		limit := minInt(10, len(rc))

		ew.println("Top failing rules (by repo count):")
		ftw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		ftwEw := &errWriter{w: ftw}
		ftwEw.println("  RULE\tFAILURES")
		ftwEw.println("  ----\t--------")
		for _, entry := range rc[:limit] {
			ftwEw.printf("  %s\t%d\n", entry.id, entry.count)
		}
		if ftwEw.err != nil {
			return ftwEw.err
		}
		if err := ftw.Flush(); err != nil {
			return err
		}
		ew.println()
	}

	// Severity breakdown
	severityBreakdown := report.SeverityBreakdown()
	if len(severityBreakdown) > 0 {
		severities := []string{"error", "warning", "info"}
		ew.println("Severity breakdown (failures only):")
		for _, sev := range severities {
			if count, ok := severityBreakdown[sev]; ok {
				ew.printf("  %-10s %d\n", sev, count)
			}
		}
		ew.println()
	}

	// Org-wide totals
	ew.printf("Org totals: %d repos scanned, %d passed, %d failed\n",
		report.RepoCount(),
		report.PassCount(),
		report.FailureCount(),
	)

	return ew.err
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
