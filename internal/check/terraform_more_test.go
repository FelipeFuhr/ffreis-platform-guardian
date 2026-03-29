package check

import (
	"testing"

	"github.com/ffreis/platform-guardian/internal/hcl"
	"github.com/ffreis/platform-guardian/internal/scanner"
)

func TestTFModuleUsedChecker_Evaluate(t *testing.T) {
	t.Parallel()

	snap := scanner.NewSnapshot("org/repo")
	snap.TFModules = []hcl.TFModule{
		{
			Path: "main.tf",
			Modules: []hcl.TFModuleCall{
				{Name: "x", Source: "github.com/acme/terraform-modules//vpc"},
			},
		},
	}

	// path.Match doesn't let '*' match '/', so exercise the "contains" fallback
	// by using a non-glob pattern.
	c := &TFModuleUsedChecker{Source: "terraform-modules"}
	got := c.Evaluate(snap)
	if got.Status != Pass {
		t.Fatalf("expected Pass, got %s: %s", got.Status, got.Message)
	}
}
