package scanner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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
