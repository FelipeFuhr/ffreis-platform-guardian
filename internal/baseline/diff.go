package baseline

import (
	"time"

	"github.com/ffreis/platform-guardian/internal/engine"
)

// FilterNew returns only results that are NOT present in the baseline as failures.
func FilterNew(report *engine.ScanReport, b *Baseline) *engine.ScanReport {
	// Build lookup map of existing baseline failures
	existing := make(map[string]bool)
	for _, entry := range b.Entries {
		if entry.Status == "fail" {
			key := entry.Repo + "/" + entry.RuleID
			existing[key] = true
		}
	}

	filtered := &engine.ScanReport{
		RunID:       report.RunID,
		GeneratedAt: report.GeneratedAt,
	}

	for _, result := range report.Results {
		if result.Status == engine.StatusFail && result.Rule != nil {
			key := result.Repo + "/" + result.Rule.ID
			if existing[key] {
				// Already in baseline — skip
				continue
			}
		}
		filtered.Results = append(filtered.Results, result)
	}

	return filtered
}

// Update merges the current report into the baseline.
// Adds new failing entries, removes entries that now pass.
func Update(b *Baseline, report *engine.ScanReport) *Baseline {
	currentFails, currentPasses := currentResultKeys(report)

	newEntries, existingKeys := retainUnresolved(b, currentPasses)
	newEntries = append(newEntries, newFailures(report, currentFails, existingKeys)...)

	return &Baseline{
		GeneratedAt: b.GeneratedAt,
		Entries:     newEntries,
	}
}

func currentResultKeys(report *engine.ScanReport) (map[string]bool, map[string]bool) {
	currentFails := make(map[string]bool)
	currentPasses := make(map[string]bool)
	for _, result := range report.Results {
		if result.Rule == nil {
			continue
		}
		key := resultKey(result.Repo, result.Rule.ID)
		switch result.Status {
		case engine.StatusFail:
			currentFails[key] = true
		case engine.StatusPass:
			currentPasses[key] = true
		}
	}
	return currentFails, currentPasses
}

func retainUnresolved(b *Baseline, currentPasses map[string]bool) ([]Entry, map[string]bool) {
	var newEntries []Entry
	existingKeys := make(map[string]bool)

	for _, entry := range b.Entries {
		if entry.Status != "fail" {
			continue
		}
		key := resultKey(entry.Repo, entry.RuleID)
		if currentPasses[key] {
			continue
		}
		newEntries = append(newEntries, entry)
		existingKeys[key] = true
	}

	return newEntries, existingKeys
}

func newFailures(report *engine.ScanReport, currentFails, existingKeys map[string]bool) []Entry {
	var newEntries []Entry
	for _, result := range report.Results {
		if result.Rule == nil || result.Status != engine.StatusFail {
			continue
		}
		key := resultKey(result.Repo, result.Rule.ID)
		if !currentFails[key] || existingKeys[key] {
			continue
		}
		newEntries = append(newEntries, Entry{
			Repo:      result.Repo,
			RuleID:    result.Rule.ID,
			Status:    "fail",
			FirstSeen: time.Now().UTC().Format(time.RFC3339),
		})
	}
	return newEntries
}

func resultKey(repo, ruleID string) string {
	return repo + "/" + ruleID
}
