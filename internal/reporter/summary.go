package reporter

import (
	"fmt"
	"io"
	"sort"

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
	if err := writeSeverityBreakdown(r.w, "Failures by severity:", report.SeverityBreakdown()); err != nil {
		return err
	}

	// Top failing rules
	if err := writeTopRuleCounts(r.w, "Most violated rules:", "  RULE\tREPOS FAILING", "  ----\t-------------", report.RuleFailureCounts()); err != nil {
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

	if err := writeTable(w, "  REPO\tFAIL\tPASS", "  ----\t----\t----", func(tew *errWriter) {
		for _, repo := range failing {
			s := repoStats[repo]
			tew.printf("  %s\t%d\t%d\n", repo, s.Fail, s.Pass)
		}
	}); err != nil {
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
