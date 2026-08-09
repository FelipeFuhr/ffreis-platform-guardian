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

	// scan-fix(golangci:errcheck): route direct writes through errWriter (see
	// helpers.go) so a real write failure (e.g. a full disk) surfaces instead of
	// being silently dropped call-by-call.
	ew := &errWriter{w: r.w}
	ew.printf("Guardian Scan Summary\n")
	ew.printf("Generated: %s   Run: %s\n\n", report.GeneratedAt, report.RunID)
	ew.printf("Repos scanned : %d\n", totalRepos)
	ew.printf("Checks passed : %d\n", totalPass)
	ew.printf("Checks failed : %d\n\n", totalFail)
	if ew.err != nil {
		return ew.err
	}

	// Severity breakdown
	if err := writeSeverityBreakdown(r.w, report.SeverityBreakdown()); err != nil {
		return err
	}

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
		// scan-fix(golangci:errcheck): propagate the write error.
		if _, err := fmt.Fprintln(r.w, "All checks passed."); err != nil {
			return err
		}
	}

	return nil
}

func writeSeverityBreakdown(w io.Writer, sevBreakdown map[string]int) error {
	if len(sevBreakdown) == 0 {
		return nil
	}

	// scan-fix(golangci:errcheck): route through errWriter, see Report() above.
	ew := &errWriter{w: w}
	ew.println("Failures by severity:")
	for _, sev := range []string{"error", "warning", "info"} {
		if count, ok := sevBreakdown[sev]; ok {
			ew.printf("  %-10s %d\n", sev, count)
		}
	}
	ew.println()
	return ew.err
}

func writeMostViolatedRules(w io.Writer, ruleCounts map[string]int) error {
	if len(ruleCounts) == 0 {
		return nil
	}

	rc := sortedRuleCounts(ruleCounts)
	limit := minInt(10, len(rc))

	// scan-fix(golangci:errcheck): route through errWriter, see Report() above.
	ew := &errWriter{w: w}
	ew.println("Most violated rules:")

	ftw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	ftwEw := &errWriter{w: ftw}
	ftwEw.println("  RULE\tREPOS FAILING")
	ftwEw.println("  ----\t-------------")
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
	return ew.err
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

	// scan-fix(golangci:errcheck): route through errWriter, see Report() above.
	ew := &errWriter{w: w}
	ew.printf("Repos with failures (%d):\n", len(failing))

	rtw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	rtwEw := &errWriter{w: rtw}
	rtwEw.println("  REPO\tFAIL\tPASS")
	rtwEw.println("  ----\t----\t----")
	for _, repo := range failing {
		s := repoStats[repo]
		rtwEw.printf("  %s\t%d\t%d\n", repo, s.Fail, s.Pass)
	}
	if rtwEw.err != nil {
		return rtwEw.err
	}
	if err := rtw.Flush(); err != nil {
		return err
	}

	ew.println()
	return ew.err
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
