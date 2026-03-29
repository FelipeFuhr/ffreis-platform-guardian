package baseline

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")

	in := &Baseline{
		Entries: []Entry{
			{Repo: "org/repo", RuleID: "rule-1", Status: "fail", FirstSeen: "2024-01-01T00:00:00Z"},
		},
	}

	if err := Save(path, in); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Ensure file is not world-readable (best-effort check; not all platforms enforce)
	if fi, err := os.Stat(path); err == nil {
		if fi.Mode().Perm()&0o077 != 0 {
			t.Fatalf("expected permissions <= 0600, got %o", fi.Mode().Perm())
		}
	}

	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(out.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(out.Entries))
	}
	if out.Entries[0].Repo != in.Entries[0].Repo {
		t.Fatalf("expected repo %q, got %q", in.Entries[0].Repo, out.Entries[0].Repo)
	}

	if out.GeneratedAt == "" {
		t.Fatalf("expected GeneratedAt to be set")
	}
	if _, err := time.Parse(time.RFC3339, out.GeneratedAt); err != nil {
		t.Fatalf("expected GeneratedAt to be RFC3339, got %q: %v", out.GeneratedAt, err)
	}
}
