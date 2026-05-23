package main

import (
	"os"
	"testing"
)

func TestMainEntryPoint_HelpDoesNotExit(t *testing.T) {
	t.Parallel()

	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })

	// Ensure Cobra doesn't see `go test` args (which would make cmd.Execute() call os.Exit(1)).
	os.Args = []string{"platform-guardian", "--help"}

	main()
}
