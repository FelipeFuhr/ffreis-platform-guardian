package cmd

import "testing"

func TestBuildLogger_InvalidLevelErrors(t *testing.T) {
	t.Parallel()

	if _, err := buildLogger("not-a-level"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestBuildLogger_ValidLevelSucceeds(t *testing.T) {
	t.Parallel()

	if _, err := buildLogger("info"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestGetLogger_FallsBackWhenMissing(t *testing.T) {
	t.Parallel()

	l := getLogger(rootCmd)
	if l == nil {
		t.Fatalf("expected logger")
	}
}
