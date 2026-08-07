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
