package scanner

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/ffreis/platform-guardian/internal/hcl"
)

// TerraformScanner clones the repo and parses all .tf files.
type TerraformScanner struct {
	snapshot *RepoSnapshot
}

func NewTerraformScanner(snap *RepoSnapshot) *TerraformScanner {
	return &TerraformScanner{snapshot: snap}
}

func (s *TerraformScanner) Type() ScannerType {
	return ScannerTypeTerraform
}

func (s *TerraformScanner) Scan(ctx context.Context, token, repo string) error {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found in PATH: install git to use terraform scanning")
	}

	tmpDir, err := os.MkdirTemp("", "platform-guardian-tf-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cloneURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", token, repo)
	if token == "" {
		cloneURL = fmt.Sprintf("https://github.com/%s.git", repo)
	}

	// Use []string args to avoid shell expansion
	cmd := exec.CommandContext(ctx,
		"git", "clone", "--depth", "1", "--quiet", cloneURL, tmpDir,
	)
	cmd.Env = os.Environ()

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, string(out))
	}

	modules, err := hcl.Walk(tmpDir)
	if err != nil {
		return fmt.Errorf("walking terraform files: %w", err)
	}

	s.snapshot.TFModules = modules
	return nil
}
