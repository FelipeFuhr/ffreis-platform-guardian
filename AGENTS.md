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
