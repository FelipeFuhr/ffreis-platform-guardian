# Agent Context

**This repo:** `ffreis-platform-guardian` — repository standards checker for the ffreis
platform. Scans GitHub repos against platform governance rules (branch protection,
required labels, workflow conventions, etc.).

## Non-obvious facts

- **Operates via GitHub API.** Org-wide scans require a token with `read:org` and
  `repo` scopes. CI uses a narrower token.

- **Exit codes:** 0 = all checks pass, 1 = check failures or errors.

- **Three main commands:** `check` (single repo hygiene), `validate` (policy rules),
  `scan-org` (org-wide scan). Each has distinct output format.

## Structure

```
cmd/              ← Cobra CLI entry point
internal/         ← checker implementations
.github/workflows/
```

## Build/run

```bash
make build
./bin/platform-guardian scan-org --org FelipeFuhr --token $GH_TOKEN
./bin/platform-guardian check --repo FelipeFuhr/some-repo
```

## Repo-wide lint/nakedgo debt — fixed (as of the scorecards.yml-removal PR, 2026-08-09)

`make lint` (72 findings: errcheck 55, gosec 6, staticcheck 7, copyloopvar 3,
goimports 1) and `make nakedgo` (1 finding, `internal/org/worker.go`), both
documented broken repo-wide since PR #57 (2026-07-31), are now clean. Root
causes and fixes, by category:

- **errcheck (55):** genuine unchecked-error findings, fixed at the root —
  `defer resp.Body.Close()` / `os.RemoveAll(tmpDir)` now use
  `defer func() { _ = x() }()` with an inline reason (best-effort cleanup);
  `internal/reporter/summary.go` + `table.go` now route every direct write
  through a small `errWriter` helper (`internal/reporter/helpers.go`) that
  accumulates the first real write error and returns it — this also fixed two
  previously-silent `_ = rtw.Flush()`/`_ = ftw.Flush()` calls in `table.go`
  that were dropping genuine flush errors; `cmd/scan_org.go`'s
  `writeJSONReport` now captures a deferred `Close()` error via a named
  return, since a `Close()` failure on a write-opened file can mean an unflushed
  write. Best-effort diagnostic writes (warnings sinks, CLI stdout/stderr void
  methods) use an explicit `_, _ = fmt.Fprintf(...)` discard with a comment.
- **staticcheck (7):** QF1012 (`sb.WriteString(fmt.Sprintf(...))` →
  `fmt.Fprintf(sb, ...)`) and QF1003 (if/else-if severity chains → tagged
  `switch`) — straightforward mechanical fixes.
- **goimports (1):** `cmd/exit_paths_test.go` had a third-party import
  (`github.com/spf13/cobra`) grouped after the local-prefix
  (`github.com/ffreis/...`) group; `.golangci.yml` sets
  `goimports.local-prefixes: [github.com/ffreis]`, so third-party imports must
  sort first.
- **copyloopvar (3):** three test files had a redundant `tc := tc` loop-var
  copy; dropped — `go.mod` already declares `go 1.25.8`, so Go 1.22+
  per-iteration loop semantics already apply.
- **gosec (6):** genuine judgment call, not a blanket suppression. Two parts:
  - `internal/hcl/walker.go` (`Walk`) now uses `os.OpenRoot`/`Root.ReadFile`
    (Go 1.24+) instead of `filepath.WalkDir` + `os.ReadFile(absPath)`. `dir` is
    a freshly `git clone`d, potentially untrusted repo (see
    `scanner.TerraformScanner.Scan`); `os.Root` resolves every path via
    openat-family syscalls and rejects any symlink that would escape `dir`,
    atomically — closing a real (if narrow) local-file-read risk from a
    maliciously named symlink in a scanned repo. This is a genuine security
    fix, not a suppression.
  - The remaining G304 (`os.Create`/`os.ReadFile` with a variable path) and
    G204 (`exec.CommandContext` with a variable) findings are a systemic,
    intentional pattern for a local CLI tool: every G304 path is either an
    operator-supplied CLI flag (`--output-file`, `--baseline`,
    `--write-baseline`, rule-dir paths) or the `os.Root`-scoped walk above;
    both G204 sites in `internal/scanner/terraform.go` invoke `git` (resolved
    via `exec.LookPath`) through a safe argv array, never a shell string. Per
    workspace convention ("prefer ONE documented gosec exclude-rule... over
    scattering suppressions"), this is now `.golangci.yml`'s
    `linters.settings.gosec.excludes: [G304, G204]`, with the reasoning
    documented inline there — read that comment before adding a new
    `os.ReadFile`/`exec.Command` call and assuming it's automatically exempt;
    re-justify per call site if the trust model ever changes (e.g. a future
    flag that accepts a path from a remote/network source).

Also fixed in the same PR: `scripts/hooks/check_required_tools.sh` was
deleted repo-wide in commit `a04a251` (migrating hook scripts to be sourced
from `ffreis-platform-standards` via lefthook `remotes:` instead of committed
locally) — but PR #63 (`feat(ci): enforce 75% coverage floor...`) later added
`scripts/hooks/check_coverage_gate.sh` and
`check_integration_coverage_gate.sh`, both of which call
`"${SCRIPT_DIR}/check_required_tools.sh" go` directly, without re-adding it.
`make coverage-gate` therefore failed with exit 127 ("No such file or
directory") on every invocation. Restored the file (same content it had
before `a04a251` deleted it — note this is *not* the copy currently vendored
in `ffreis-platform-standards`, which has a `return "$missing"` at the top
level that only works when sourced, not executed directly; this repo's copy
uses `exit "$missing"`, correct for direct execution) and force-added it since
`scripts/hooks/` is `.gitignore`d as a whole (its two siblings were already
tracked the same way). If `ffreis-platform-standards` ever fixes/republishes
this script such that `make coverage-gate`/`make integration-coverage-gate`
can source it instead of relying on a locally committed copy, prefer that and
delete this repo's copy — don't let the two drift.

**`pre-commit-compat` is also currently broken** (discovered 2026-08-06): this
repo's `.pre-commit-config.yaml` pins `golangci-lint` to `v1.52.2`, but the
locally installed binary is v2.x. `.golangci.yml`'s `version: "2"` key is only
valid under golangci-lint v2's config schema — v1 hard-crashes trying to parse
it (`Can't read config: ... 'Version' expected a map, got 'string'`), so this
hook fails on **every** commit regardless of content. Requires
`LEFTHOOK_EXCLUDE=lint,pre-commit-compat` until fixed. Not fixed inline because
bumping the pin to v2 changes golangci-lint's default linter set and will
likely surface a new wave of findings needing their own triage — do that as
its own scoped PR, same as the `lint`/`nakedgo` debt above.

## Reusable-workflow permission contract (fixed in PR #60, 2026-08-06)

`ci.yml`, `devops-go-ci.yml`, and `devops-pr-hygiene.yml` all had
`conclusion: startup_failure` with **zero jobs ever created** on every run
from 2026-05-23/24 until PR #60 — GitHub's generic "workflow file issue"
message, with `actionlint` passing clean and every `uses:` SHA resolving
fine, so it looked unrelated to any local mistake. Root cause: **a calling
job's `permissions:` block must grant at least what the called reusable
workflow's job requires** — either the called job's own `permissions:`
override, or the reusable workflow's top-level `permissions:` default when
the job has no override. If the caller under-grants, GitHub rejects the
**entire run** at parse time (not just that one job) — job creation for a
run is atomic, so one under-permissioned job takes every other job down
with it. Two jobs were under-granting here: `lint` (needed
`pull-requests: read` for `golangci-lint-action`'s PR lookup, only had
`contents: read`) and `semantic-pr` (needed `contents: read`, inherited
from `general-semantic-pr.yml`'s top-level default since that job has no
override; only had `pull-requests: read`).

**When adding a job that calls ANY `FelipeFuhr/ffreis-workflows-*` reusable
workflow, check the callee's OWN `permissions:` (both job-level and
top-level-as-fallback) and mirror the full set in the caller's job-level
`permissions:` block** — do not guess from what the job "seems" to need
logically (that's exactly how PR #23 broke `semantic-pr`: it looked
correct to grant only `pull-requests: read` to a "check the PR title" job).
`actionlint` and cross-repo `uses:`/SHA resolution checks do **not** catch
this class of bug — it's a runtime permission-contract mismatch, not a
syntax or resolution error.

**Recurred 2026-08-09 (fixed same PR as the lint/nakedgo debt above):** the
PR #63 pin bump of `FelipeFuhr/ffreis-workflows-go` to `v1.3.0` carried a new
`id-token: write` requirement on `go-test.yml`'s `test` job (added for
optional L2 S3 build-cache OIDC support — a no-op at runtime since
`go-cache-s3` defaults `false`, but GitHub still validates the permission
contract statically at parse time). `ci.yml`'s and `devops-go-ci.yml`'s
`test:` jobs only granted `contents: read`, so both workflows went back to
zero-job `startup_failure` on every run. **A reusable-workflow SHA/tag pin
bump is itself a trigger to re-diff the callee's job `permissions:` against
what every caller job in this repo grants — not just when adding a brand new
job.** The failure is silent until the next PR: draft PRs still show "no
checks" (this class of failure doesn't distinguish itself from the normal
zero-CI-on-draft state), so it isn't caught until promotion to ready, and
`gh pr checks` reports it as perpetual PENDING — only `check-ci.sh` (or
`gh run view <run-id>`, which prints "This run likely failed because of a
workflow file issue") surfaces it.

## `check` stdout is report-only (fixed 2026-08-06)

`platform-guardian check --format sarif|json|annotations` writes exactly the
formatted report to stdout and nothing else — CI redirects it straight to a
file (`> guardian-report.sarif`) for `github/codeql-action/upload-sarif` to
consume. `runCheck` (`cmd/check.go`) previously wrote a human status line
("[ok] check completed...") to the *same* stdout stream on the success path
(the failure path already correctly used stderr via `ErrStatus`), which
appended plain text after the closing `}` of the JSON document. This was
invisible for years because a separate, since-fixed `FLEET_READ_TOKEN` bug
made every scanner call 401 before enough SARIF content existed to make the
corruption reachable — once the token was fixed and the self-scan started
succeeding, `upload-sarif` began failing with "Invalid SARIF. JSON syntax
error: Unexpected non-whitespace character after JSON".

**Any status/log text `runCheck` (or a future reporter) emits must go to
`cmd.ErrOrStderr()`, never `cmd.OutOrStdout()`** — stdout is reserved
exclusively for the reporter's output, for every format, not just the
machine-readable ones. `cmd/check_cmd_test.go`'s
`TestRunCheck_SarifFormat_StdoutIsSingleValidJSONDocument` guards this by
decoding the CLI's captured stdout and asserting there is no trailing content
after the JSON value; it fails immediately if this regresses.

## Public repo — private-repo hygiene

This is a **public** GitHub repository. When writing commit messages, PR titles,
PR descriptions, or any other user-visible text, **never name private repos** —
website content, inventory, infra, Lambda, or data repos that are not publicly
listed. Use generic terms instead: "the fleet inventory", "a private consumer",
"internal infra", "private data repo", etc.

## Keeping this file current

- **If you discover a fact not reflected here:** add it before finishing your task.
- **If something here is wrong or outdated:** correct it in the same commit as the code change.
- **If you rename a file, command, or concept referenced here:** update the reference.
