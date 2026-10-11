#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

if [[ "$#" -eq 0 ]]; then
  echo "Usage: $0 <tool> [tool...]" >&2
  exit 1
fi

missing=0

install_hint() {
  local tool_name="$1"
  case "$tool_name" in
    golangci-lint)
      echo "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
      ;;
    gitleaks)
      echo "https://github.com/gitleaks/gitleaks#installing"
      ;;
    govulncheck)
      echo "go install golang.org/x/vuln/cmd/govulncheck@latest"
      ;;
    gofmt)
      echo "Install Go from https://go.dev/dl/"
      ;;
    go)
      echo "Install Go from https://go.dev/dl/"
      ;;
    *)
      echo "Install '$tool_name' and ensure it is available in PATH."
      ;;
  esac
  return 0
}

for tool in "$@"; do
  local_tool="$tool"
  if command -v "$local_tool" >/dev/null 2>&1; then
    continue
  fi

  echo "Missing required tool: $local_tool" >&2
  echo "Install hint: $(install_hint "$local_tool")" >&2
  missing=1
done

# scan-fix(bash:top-level-return): `exit`, not `return` — this script is
# EXECUTED by consumers (e.g. `./scripts/hooks/check_required_tools.sh $(GOFMT)`
# in a Makefile), never sourced. `return` outside a function/sourced script is
# a hard bash error ("return: can only `return' from a function or sourced
# script"), so every invocation died before checking anything — the fmt/lint
# gate silently never ran. Confirmed no fleet consumer sources this file
# (all invoke it as `./scripts/hooks/check_required_tools.sh` or
# `"${SCRIPT_DIR}/check_required_tools.sh"`), so `exit` is safe everywhere.
exit "$missing"
