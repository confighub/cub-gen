# Connected CI Bootstrap

Use this checklist to get the first green `CI Connected` run and keep it reliable.

## 1) Configure required secrets/vars

The connected workflow enforces Story 10 proof inputs and strict no-fallback policy.

Required secrets:

- `CONFIGHUB_BASE_URL`
- `CONFIGHUB_SPACE`
- `CONFIGHUB_TOKEN`
- `GH_TOKEN` (or set `GITHUB_TOKEN` with equivalent repo-read scope)

Required repository variables (or secrets):

- `APP_PR_REPO` (for example: `confighub/cub-gen`)
- `APP_PR_NUMBER`
- `PROMOTION_PR_REPO`
- `PROMOTION_PR_NUMBER`

Notes:

- `APP_PR_*` and `PROMOTION_PR_*` must point to real PRs whose commits are signed/verified and whose target branches enforce protection.
- For private repos, ensure `GH_TOKEN` can read PR, commit verification metadata, and branch protection settings.

## 2) Trigger and verify

Run the `CI` workflow from GitHub Actions (push/PR/workflow_dispatch). The `connected` job runs only when ConfigHub secrets are present.

Expected strict behavior:

- `CONNECTED_FALLBACK_MODE=off`
- `ALLOW_FALLBACK_INGEST=0`
- `ALLOW_STORY_10_SKIP=0`

If any required Story 10 input is missing, `CI Connected` fails at the preflight step.

## 3) Branch protection recommendation

Set required checks on your protected branch:

1. `CI Local`
2. `CI Connected` (for internal PR lanes where secrets are available)

For external fork PRs (where secrets are unavailable), use a maintainer rerun policy or a separate trusted promotion lane.

## 4) Troubleshooting lane (non-release)

Use only for diagnostics, never for release qualification:

```bash
make ci-connected-troubleshoot
```

This explicitly enables:

- `CONNECTED_FALLBACK_MODE=changeset`
- `ALLOW_FALLBACK_INGEST=1`
- `ALLOW_STORY_10_SKIP=1`

## 5) PR DRY ownership gate (WET edit blocker)

Use the dedicated workflow:

- `.github/workflows/pr-dry-ownership-gate.yml`

What it does:

- Runs `test/checks/pr-dry-ownership-gate.sh` for:
  - `examples/helm-paas`
  - `examples/springboot-paas`
- Compares PR-changed YAML/JSON files to recognized DRY inputs.
- Blocks merge if a PR edits non-DRY/WET paths (or wrong-owner DRY paths when actor role is enforced).
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
