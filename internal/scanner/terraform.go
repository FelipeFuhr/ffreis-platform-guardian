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

	cloneURL := fmt.Sprintf("https://github.com/%s.git", repo)
	if token != "" {
		cloneURL = fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", token, repo)
	}

	ref := s.snapshot.Ref
	if ref == "" {
		// Default behavior: shallow clone default branch.
		cmd := exec.CommandContext(ctx,
			"git", "clone", "--depth", "1", "--quiet", cloneURL, tmpDir,
		)
		cmd.Env = os.Environ()
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git clone failed: %w\n%s", err, string(out))
		}
	} else {
		// Clone the specific ref (commit SHA or ref name) to match the CI checkout.
		initCmd := exec.CommandContext(ctx, "git", "init", "--quiet", tmpDir)
		initCmd.Env = os.Environ()
		if out, err := initCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git init failed: %w\n%s", err, string(out))
		}

		remoteCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "remote", "add", "origin", cloneURL)
		remoteCmd.Env = os.Environ()
		if out, err := remoteCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git remote add failed: %w\n%s", err, string(out))
		}

		fetchCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "fetch", "--depth", "1", "--quiet", "origin", ref)
		fetchCmd.Env = os.Environ()
		if out, err := fetchCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git fetch %s failed: %w\n%s", ref, err, string(out))
		}

		checkoutCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "checkout", "--quiet", "FETCH_HEAD")
		checkoutCmd.Env = os.Environ()
		if out, err := checkoutCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git checkout %s failed: %w\n%s", ref, err, string(out))
		}
	}

	modules, err := hcl.Walk(tmpDir)
	if err != nil {
		return fmt.Errorf("walking terraform files: %w", err)
	}

	s.snapshot.TFModules = modules
	return nil
}
