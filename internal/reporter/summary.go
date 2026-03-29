package reporter

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/ffreis/platform-guardian/internal/engine"
)

// SummaryReporter emits a compact org-wide aggregate report with no per-result rows.
// Suitable for CI log output when scanning large numbers of repositories.
type SummaryReporter struct {
	w io.Writer
}

func (r *SummaryReporter) Report(report *engine.ScanReport) error {
	totalRepos := report.RepoCount()
	totalPass := report.PassCount()
	totalFail := report.FailureCount()

	fmt.Fprintf(r.w, "Guardian Scan Summary\n")
	fmt.Fprintf(r.w, "Generated: %s   Run: %s\n\n", report.GeneratedAt, report.RunID)
	fmt.Fprintf(r.w, "Repos scanned : %d\n", totalRepos)
	fmt.Fprintf(r.w, "Checks passed : %d\n", totalPass)
	fmt.Fprintf(r.w, "Checks failed : %d\n\n", totalFail)

	// Severity breakdown
	writeSeverityBreakdown(r.w, report.SeverityBreakdown())

	// Top failing rules
	if err := writeMostViolatedRules(r.w, report.RuleFailureCounts()); err != nil {
		return err
	}

	// Repos with failures (sorted by failure count)
	repoStats := report.RepoSummary()
	if err := writeFailingRepos(r.w, repoStats); err != nil {
		return err
	}

	if totalFail == 0 {
		fmt.Fprintln(r.w, "All checks passed.")
	}

	return nil
}

func writeSeverityBreakdown(w io.Writer, sevBreakdown map[string]int) {
	if len(sevBreakdown) == 0 {
		return
	}

	fmt.Fprintln(w, "Failures by severity:")
	for _, sev := range []string{"error", "warning", "info"} {
		if count, ok := sevBreakdown[sev]; ok {
			fmt.Fprintf(w, "  %-10s %d\n", sev, count)
		}
	}
	fmt.Fprintln(w)
}

func writeMostViolatedRules(w io.Writer, ruleCounts map[string]int) error {
	if len(ruleCounts) == 0 {
		return nil
	}

	rc := sortedRuleCounts(ruleCounts)
	limit := minInt(10, len(rc))

	fmt.Fprintln(w, "Most violated rules:")
	ftw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(ftw, "  RULE\tREPOS FAILING")
	fmt.Fprintln(ftw, "  ----\t-------------")
	for _, entry := range rc[:limit] {
		fmt.Fprintf(ftw, "  %s\t%d\n", entry.id, entry.count)
	}
	if err := ftw.Flush(); err != nil {
		return err
	}
	fmt.Fprintln(w)
	return nil
}

func writeFailingRepos(w io.Writer, repoStats map[string]*engine.RepoStats) error {
	failing := failingRepos(repoStats)
	if len(failing) == 0 {
		return nil
	}

	sort.Slice(failing, func(i, j int) bool {
		fi := repoStats[failing[i]].Fail
		fj := repoStats[failing[j]].Fail
		if fi != fj {
			return fi > fj
		}
		return failing[i] < failing[j]
	})

	fmt.Fprintf(w, "Repos with failures (%d):\n", len(failing))
	rtw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(rtw, "  REPO\tFAIL\tPASS")
	fmt.Fprintln(rtw, "  ----\t----\t----")
	for _, repo := range failing {
		s := repoStats[repo]
		fmt.Fprintf(rtw, "  %s\t%d\t%d\n", repo, s.Fail, s.Pass)
	}
	if err := rtw.Flush(); err != nil {
		return err
	}
	fmt.Fprintln(w)
	return nil
}

func failingRepos(repoStats map[string]*engine.RepoStats) []string {
	var failing []string
	for repo, s := range repoStats {
		if s.Fail > 0 {
			failing = append(failing, repo)
		}
	}
	return failing
}
