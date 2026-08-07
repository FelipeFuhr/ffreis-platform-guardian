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

## Known repo-wide lint/nakedgo debt (as of PR #57, 2026-07-31)

`make lint` (72 findings: errcheck, gosec, staticcheck, copyloopvar, goimports)
and `make nakedgo` (1 finding, `internal/org/worker.go`) currently fail
repo-wide on `main`. Both are wired into the lefthook `pre-commit`/`go.yml`
`lint` command (via the `ffreis-platform-standards` remote), which runs
whole-tree, not just staged files — so **any** commit to this repo currently
gets blocked by `git commit` unless you pass `LEFTHOOK_EXCLUDE=lint` (not
`--no-verify` — every other real hook still runs). This surfaced because
lefthook had apparently never been installed locally for this repo before
(`.git/hooks/` only had `.sample` files); GitHub Actions CI is currently
billing-paused fleet-wide so it hadn't caught it either. Needs its own cleanup
PR; do not silently keep excluding `lint` forever.

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
