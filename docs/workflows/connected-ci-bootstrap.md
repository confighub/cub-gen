# Connected CI Bootstrap

Use this checklist to get the first green ConfigHub smoke run and keep the
deeper connected lane understandable.

## 1) Configure required secrets for `make ci-connected`

The release-facing connected lane is now a smaller ConfigHub smoke test.

Required secrets:

- `CONFIGHUB_BASE_URL`
- `CONFIGHUB_SPACE`
- `CONFIGHUB_TOKEN`

What it proves:

- ConfigHub credentials and space resolution work,
- `cub-gen` can run the source-side evidence pipeline for flagship examples,
- the repo has a real connected environment available for smoke qualification.

## 2) Optional inputs for `make ci-connected-deep`

The deep lane keeps the older connected stories, flow proofs, and live
reconcile checks available outside the release smoke path.

Additional secret:

- `GH_TOKEN` (or `github.token`) for release downloads and repo reads

Additional repository variables (or secrets):

- `APP_PR_REPO`
- `APP_PR_NUMBER`
- `PROMOTION_PR_REPO`
- `PROMOTION_PR_NUMBER`

Notes:

- `APP_PR_*` and `PROMOTION_PR_*` must point to real PRs whose commits are
  signed/verified and whose target branches enforce protection.
- The deep lane still depends on the bridge endpoint shape or explicit fallback
  mode.

## 3) Trigger and verify

Run the `CI` workflow from GitHub Actions (push/PR/workflow_dispatch). The
`ConfigHub Smoke` job runs only when the three ConfigHub secrets are present.

Expected smoke behavior:

- real ConfigHub auth preflight,
- `helm-paas` and `springboot-paas` smoke coverage,
- no bridge-endpoint dependency.

Run the deep lane manually in a trusted environment with:

```bash
make ci-connected-deep
```

## 4) Branch protection recommendation

Set required checks on your protected branch:

1. `CI Local`
2. `ConfigHub Smoke` (for internal PR lanes where secrets are available)

For external fork PRs (where secrets are unavailable), use a maintainer rerun
policy or a separate trusted promotion lane.

## 5) Troubleshooting lane (non-release)

Use only for deep-lane diagnostics, never for release qualification:

```bash
make ci-connected-troubleshoot
```

This explicitly enables:

- `CONNECTED_FALLBACK_MODE=changeset`
- `ALLOW_FALLBACK_INGEST=1`
- `ALLOW_STORY_10_SKIP=1`

## 6) PR DRY ownership gate (WET edit blocker)

Use the dedicated workflow:

- `.github/workflows/pr-dry-ownership-gate.yml`

What it does:

- Runs `test/checks/pr-dry-ownership-gate.sh` for:
  - `examples/helm-paas`
  - `examples/springboot-paas`
- Compares PR-changed YAML/JSON files to recognized DRY inputs.
- Blocks merge if a PR edits non-DRY/WET paths (or wrong-owner DRY paths when
  actor role is enforced).
- Posts an actionable PR comment with:
  - `wet_path`
  - suggested `dry_path`
  - suggested DRY file candidates
  - owner
  - confidence

Manual run (local against refs):

```bash
./test/checks/pr-dry-ownership-gate.sh ./examples/helm-paas origin/main HEAD app-team --report-json .tmp/pr-gate/helm.json
./test/checks/pr-dry-ownership-gate.sh ./examples/springboot-paas origin/main HEAD app-team --report-json .tmp/pr-gate/spring.json
```
