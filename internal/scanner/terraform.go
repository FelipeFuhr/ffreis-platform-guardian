package scanner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ffreis/platform-guardian/internal/hcl"
)

const fixedPathEnv = "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
const gitQuietFlag = "--quiet"

func safeExecEnv() []string {
	env := os.Environ()
	safe := env[:0]
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			continue
		}
		safe = append(safe, kv)
	}
	safe = append(safe, fixedPathEnv)
	return safe
}

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

func lookPathGit() (string, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return "", fmt.Errorf("git not found in PATH: install git to use terraform scanning")
	}
	return gitPath, nil
}

func (s *TerraformScanner) cloneRepo(ctx context.Context, gitPath string, env []string, authArgs []string, cloneURL, tmpDir string) error {
	ref := s.snapshot.Ref
	if ref == "" {
		return shallowClone(ctx, gitPath, env, authArgs, cloneURL, tmpDir)
	}
	return cloneAtRef(ctx, gitPath, env, authArgs, cloneURL, tmpDir, ref)
}

func shallowClone(ctx context.Context, gitPath string, env []string, authArgs []string, cloneURL, tmpDir string) error {
	args := append(authArgs, "clone", "--depth", "1", gitQuietFlag, cloneURL, tmpDir)
	cmd := exec.CommandContext(ctx, gitPath, args...)
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, string(out))
	}
	return nil
}

func cloneAtRef(ctx context.Context, gitPath string, env []string, authArgs []string, cloneURL, tmpDir, ref string) error {
	// Clone the specific ref (commit SHA or ref name) to match the CI checkout.
	if out, err := runGit(ctx, gitPath, env, "init", gitQuietFlag, tmpDir); err != nil {
		return fmt.Errorf("git init failed: %w\n%s", err, out)
	}
	if out, err := runGit(ctx, gitPath, env, "-C", tmpDir, "remote", "add", "origin", cloneURL); err != nil {
		return fmt.Errorf("git remote add failed: %w\n%s", err, out)
	}
	fetchArgs := append(authArgs, "-C", tmpDir, "fetch", "--depth", "1", gitQuietFlag, "origin", ref)
	if out, err := runGit(ctx, gitPath, env, fetchArgs...); err != nil {
		return fmt.Errorf("git fetch %s failed: %w\n%s", ref, err, out)
	}
	if out, err := runGit(ctx, gitPath, env, "-C", tmpDir, "checkout", gitQuietFlag, "FETCH_HEAD"); err != nil {
		return fmt.Errorf("git checkout %s failed: %w\n%s", ref, err, out)
	}
	return nil
}

func runGit(ctx context.Context, gitPath string, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, gitPath, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (s *TerraformScanner) Scan(ctx context.Context, token, repo string) error {
	gitPath, err := lookPathGit()
	if err != nil {
		return err
	}
	env := safeExecEnv()

	tmpDir, err := os.MkdirTemp("", "platform-guardian-tf-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cloneURL := fmt.Sprintf("https://github.com/%s.git", repo)
	// Never embed the token in the clone URL — it can leak via process listings
	// and git error output.  Pass auth via git's http.extraheader config option instead.
	authArgs := gitAuthArgs(token)

	if err := s.cloneRepo(ctx, gitPath, env, authArgs, cloneURL, tmpDir); err != nil {
		return err
	}

	modules, err := hcl.Walk(tmpDir)
	if err != nil {
		return fmt.Errorf("walking terraform files: %w", err)
	}

	s.snapshot.TFModules = modules
	return nil
}

// gitAuthArgs returns the global git flags needed to authenticate via token
// without embedding it in the clone URL.  Passing auth via http.extraheader
// avoids credential exposure in process listings, git remotes, and error logs.
// Returns nil when token is empty (unauthenticated / public repos).
func gitAuthArgs(token string) []string {
	if token == "" {
		return nil
	}
	return []string{"-c", "http.https://github.com/.extraheader=AUTHORIZATION: bearer " + token}
}
