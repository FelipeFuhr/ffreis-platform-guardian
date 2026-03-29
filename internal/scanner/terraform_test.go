package scanner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTerraformScanner_Scan_WithFakeGit_ShallowClone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake git helper uses sh")
	}

	dir := t.TempDir()
	gitPath := filepath.Join(dir, "git")
	if err := os.WriteFile(gitPath, []byte(`#!/bin/sh
set -eu
cmd="$1"
case "$cmd" in
  clone)
    dest="$6"
    mkdir -p "$dest"
    cat > "$dest/main.tf" <<'EOF'
terraform {
  required_providers {
    aws = { source = "hashicorp/aws", version = ">= 5.0.0" }
  }
}
EOF
    exit 0
    ;;
  init)
    mkdir -p "$3"
    exit 0
    ;;
  -C)
    workdir="$2"
    mkdir -p "$workdir"
    if [ ! -f "$workdir/main.tf" ]; then
      echo 'terraform {}' > "$workdir/main.tf"
    fi
    exit 0
    ;;
esac
exit 0
`), 0o700); err != nil {
		t.Fatalf("write fake git: %v", err)
	}

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", origPath) })
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+origPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}

	snap := NewSnapshot("org/repo")
	s := NewTerraformScanner(snap)
	if err := s.Scan(context.Background(), "", "org/repo"); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(snap.TFModules) == 0 {
		t.Fatalf("expected terraform modules to be populated")
	}
}

func TestTerraformScanner_Scan_WithFakeGit_SpecificRef(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake git helper uses sh")
	}

	dir := t.TempDir()
	gitPath := filepath.Join(dir, "git")
	if err := os.WriteFile(gitPath, []byte(`#!/bin/sh
set -eu
cmd="$1"
case "$cmd" in
  init)
    mkdir -p "$3"
    exit 0
    ;;
  -C)
    workdir="$2"
    mkdir -p "$workdir"
    if [ ! -f "$workdir/main.tf" ]; then
      echo 'terraform {}' > "$workdir/main.tf"
    fi
    exit 0
    ;;
esac
exit 0
`), 0o700); err != nil {
		t.Fatalf("write fake git: %v", err)
	}

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", origPath) })
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+origPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}

	snap := NewSnapshot("org/repo")
	snap.Ref = "deadbeef"
	s := NewTerraformScanner(snap)
	if err := s.Scan(context.Background(), "", "org/repo"); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(snap.TFModules) == 0 {
		t.Fatalf("expected terraform modules to be populated")
	}
}

func TestTerraformScanner_TokenNotInCloneURL(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake git helper uses sh")
	}

	dir := t.TempDir()
	argsFile := filepath.Join(dir, "git-args.txt")
	gitPath := filepath.Join(dir, "git")

	// The fake git records all argv to a file so the test can inspect them.
	script := `#!/bin/sh
printf '%s\n' "$@" >> ` + argsFile + `
# Skip -c <value> so $1 always equals the git subcommand
if [ "$1" = "-c" ]; then
  shift 2
fi
cmd="$1"
case "$cmd" in
  clone)
    # After optional -c shift: clone --depth 1 --quiet <url> <dest> → dest is $6
    dest="$6"
    mkdir -p "$dest"
    echo 'terraform {}' > "$dest/main.tf"
    exit 0
    ;;
  init)
    mkdir -p "$3"
    exit 0
    ;;
  -C)
    workdir="$2"
    mkdir -p "$workdir"
    if [ ! -f "$workdir/main.tf" ]; then
      echo 'terraform {}' > "$workdir/main.tf"
    fi
    exit 0
    ;;
esac
exit 0
`
	if err := os.WriteFile(gitPath, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake git: %v", err)
	}

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", origPath) })
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+origPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}

	const secret = "supersecrettoken"
	snap := NewSnapshot("org/repo")
	s := NewTerraformScanner(snap)
	if err := s.Scan(context.Background(), secret, "org/repo"); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	recorded, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("reading recorded args: %v", err)
	}
	args := string(recorded)

	// Token must NOT appear in any URL argument.
	if strings.Contains(args, "x-access-token:"+secret) {
		t.Error("token was embedded in the clone URL — credential leak detected")
	}
	// Token MUST appear in the http.extraheader config value (the -c argument).
	if !strings.Contains(args, "AUTHORIZATION: bearer "+secret) {
		t.Error("expected token to be passed via http.extraheader, but it was not found")
	}
}
