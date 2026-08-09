package reporter

import (
	"io"
	"sort"

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
	// scan-fix(golangci:errcheck): route through errWriter (see helpers.go) so a
	// real write failure surfaces instead of being silently dropped call-by-call.
	return writeTable(w, "REPO\tRULE\tSEVERITY\tSTATUS\tMESSAGE", "----\t----\t--------\t------\t-------", func(tew *errWriter) {
		for _, result := range report.Results {
			ruleName, severity := rulePresentation(result.Rule)
			tew.printf("%s\t%s\t%s\t%s\t%s\n", result.Repo, ruleName, severity, string(result.Status), result.Message)
		}
	})
}

func rulePresentation(r *rule.Rule) (string, string) {
	if r == nil {
		return "", ""
	}
	return r.Name, string(r.Severity)
}

// writeAggregateReport delegates each section to its own helper (see below)
// so that a real write failure anywhere still short-circuits the whole
// report, without one large function accumulating every section's branching.
//
// scan-fix(sonar:S3776): originally one function inlining all four sections
// (cognitive complexity 20, over the 15 ceiling) inline; split per section.
func writeAggregateReport(w io.Writer, report *engine.ScanReport) error {
	// scan-fix(golangci:errcheck): route through errWriter (see helpers.go) so a
	// real write failure surfaces instead of being silently dropped call-by-call.
	ew := &errWriter{w: w}
	ew.println()
	ew.printf("=== Aggregate Report ===\n\n")
	if ew.err != nil {
		return ew.err
	}

	if err := writePerRepoBreakdown(w, report.RepoSummary()); err != nil {
		return err
	}
	if err := writeTopRuleCounts(w, "Top failing rules (by repo count):", "  RULE\tFAILURES", "  ----\t--------", report.RuleFailureCounts()); err != nil {
		return err
	}
	if err := writeSeverityBreakdown(w, "Severity breakdown (failures only):", report.SeverityBreakdown()); err != nil {
		return err
	}

	// Org-wide totals
	ew.printf("Org totals: %d repos scanned, %d passed, %d failed\n",
		report.RepoCount(),
		report.PassCount(),
		report.FailureCount(),
	)
	return ew.err
}

func writePerRepoBreakdown(w io.Writer, repoStats map[string]*engine.RepoStats) error {
	repos := sortedReposByFailures(repoStats)
	if len(repos) == 0 {
		return nil
	}

	ew := &errWriter{w: w}
	ew.println("Per-repo breakdown:")
	if err := writeTable(w, "  REPO\tPASS\tFAIL\tSKIP\tERROR", "  ----\t----\t----\t----\t-----", func(tew *errWriter) {
		for _, repo := range repos {
			s := repoStats[repo]
			tew.printf("  %s\t%d\t%d\t%d\t%d\n", repo, s.Pass, s.Fail, s.Skip, s.Error)
		}
	}); err != nil {
		return err
	}
	ew.println()
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
