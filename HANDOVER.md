# Session Handover — cub-gen Generator Visibility

**Date**: 2026-03-08
**Repo**: `github.com/confighub/cub-gen`
**Branch**: `main` (up to date, PRs #109, #110, #112, and #113 merged)

---

## What was completed in previous sessions

### 1. Two new generator families added: c3agent + swamp

Full implementation across all layers:

| Layer | File | What was added |
|---|---|---|
| Types | `internal/model/types.go` | `GeneratorC3Agent`, `GeneratorSwamp` constants |
| Registry | `internal/registry/registry.go` | Two complete `FamilySpec` entries (~120 lines each) |
| Detection | `internal/detect/detect.go` | `detectC3Agent()` and `detectSwamp()` wired into `ScanRepo()` |
| Import | `internal/importer/importer.go` | c3agent + swamp cases in 4 switch blocks + 2 helper functions |
| Examples | `examples/c3agent/` | `c3agent.yaml` + `c3agent-prod.yaml` |
| Examples | `examples/swamp-automation/` | `.swamp.yaml` + `workflow-deploy.yaml` |
| Tests | `cmd/cub-gen/gitops_parity_test.go` | 8 golden tests (discover/import x2, publish x4) |
| Tests | `internal/contracts/triple_conformance_fixtures_test.go` | Fixture entries for both |
| Tests | `internal/registry/registry_test.go` | Expected kinds list updated |
| Tests | `cmd/cub-gen/examples_matrix_test.go` | Bridge symmetry entries |
| Tests | `cmd/cub-gen/publish_parity_test.go` | 4 publish golden tests |

**PR #109** — shipped and merged. CI green.

### 2. Merged jesper-ai-cloud → c3agent, made both generators visible

- Deleted `examples/jesper-ai-cloud/` (was backstage shim for same project)
- Renamed demo scripts: `scenario-1-jesper-ai-cloud.sh` → `scenario-1-c3agent.sh`, `scenario-2-swamp-project.sh` → `scenario-2-swamp.sh`
- Updated all READMEs: root, examples, demo, ai-work-platform
- Updated story cards in `docs/agentic-gitops/03-worked-examples/04-eight-example-story-cards.md`
- Added c3agent + swamp to quickstart, publish, verify, attest blocks in root README

**PR #110** — shipped and merged. CI green.

### 3. Fixed swamp path semantics + expanded details payload + added Markdown introspection

Delivered in two follow-up PRs:

- **PR #112** (`fix(swamp)...` + details payload expansion):
  - Corrected swamp model-binding dry path to `jobs[].steps[].task.modelIdOrName`.
  - Added nested swamp workflow detection support (`workflow-*.yaml|yml` in child directories).
  - Expanded `cub-gen generators --json --details` to include full policy/provenance templates (role rules/defaults, hint defaults, inverse reasons/hints, wet targets, lineage templates, transform labels).
  - Updated parity goldens + docs (`README`, roadmap, release notes).
- **PR #113** (`feat(generators): add markdown introspection output`):
  - Added `cub-gen generators --markdown`.
  - Added `cub-gen generators --markdown --details`.
  - Added guardrails:
    - `--markdown` cannot be combined with `--json`.
    - `--details` requires `--json` or `--markdown`.
  - Added parity tests + golden contracts for Markdown output.

---

## Current state of main

- 8 generator families: helm, score, springboot, backstage, ably, opsworkflow, **c3agent**, **swamp**
- All tests pass (`make ci` green)
- All 4 AI work platform demo scenarios pass
- Golden files generated for all generators
- Platform-owner visibility now available via:
  - `cub-gen generators --json --details`
  - `cub-gen generators --markdown --details`

---

## What needs to happen next

### Problem statement

The generator "triple" (contract + provenance + inverse transform plan) is the core intellectual property of each generator — it defines:
- What DRY files are consumed and what WET manifests are produced
- How each field maps from DRY→WET with confidence scores
- Who owns each field (app-team vs platform-engineer)
- Which changes need review
- How to edit WET back to DRY

The triple source-of-truth is still Go struct literals in `internal/registry/registry.go`. Visibility is now much better (`--json --details` and `--markdown --details`), but authoring still requires Go edits.

### Two personas with different needs

| Persona | Need | Current experience |
|---|---|---|
| **Platform owner** | See the mapping, check it, modify it, create new generators | Can inspect rich Markdown/JSON details; still needs Go to author |
| **Application owner** | "Magic, it just works" | Existing wizard + import/publish pipeline is fine |

### Three approaches were designed (see plan file)

The plan file at `.claude/plans/snappy-frolicking-blossom.md` contains concrete, worked examples of all three approaches using c3agent as the specimen:

**Approach A — YAML-first registry**
- Move generator definitions from Go struct literals to `generators/c3agent.yaml` (one file per generator)
- Go loads YAML via `embed.FS` at init
- Platform owners read/edit/copy YAML files
- New generator = new YAML file (no Go changes for the triple)
- ✅ Platform eng lingua franca, ⚠️ detection logic still requires Go

**Approach B — Generated docs + diagrams**
- Keep registry in Go, auto-generate markdown + Mermaid per generator
- `docs/generators/c3agent.md` with data flow diagrams, ownership tables, field maps
- CI-enforced freshness (`go generate ./...` + git diff check)
- ✅ Browsable on GitHub, ✅ additive only, ⚠️ creating new generators still requires Go

**Approach C — YAML files + generated docs**
- YAML files are source of truth (Approach A)
- Docs auto-generated from YAML (Approach B)
- ✅ Best of both, ⚠️ most implementation effort

### User's decision and where we are now

The user approved the plan with all three approaches documented. They want to **evaluate all three with concrete examples** before choosing. The plan file contains full worked specimens of each approach.

Status after PR #113:

- We now have an operational slice of **Approach B** (readability/inspection).
- We have not yet started **Approach A** (YAML-first authoring) or **Approach C** (YAML + generated docs).

---

## Key files for the next session

### Source of truth (today)

| File | Purpose |
|---|---|
| `internal/registry/registry.go` | All 8 generator FamilySpec definitions (~510 lines of struct literals) |
| `internal/model/types.go` | GeneratorKind type + constants |
| `internal/detect/detect.go` | Detection functions (1 per generator) |
| `internal/importer/importer.go` | Import logic with switch statements |

### CLI entry point

| File | Purpose |
|---|---|
| `cmd/cub-gen/main.go` | All CLI commands. `runGenerators()` is the closest existing command to an "explain" view |

### Test infrastructure

| File | Purpose |
|---|---|
| `cmd/cub-gen/gitops_parity_test.go` | Golden file tests for all commands |
| `cmd/cub-gen/testdata/parity/` | Golden JSON + text files |
| `internal/contracts/triple_conformance_fixtures_test.go` | Bridge symmetry (every kind needs a fixture) |
| `internal/registry/registry_test.go` | Registry completeness assertions |

### Example repos

| Directory | Generator | Key files |
|---|---|---|
| `examples/c3agent/` | c3agent | `c3agent.yaml`, `c3agent-prod.yaml` |
| `examples/swamp-automation/` | swamp | `.swamp.yaml`, `workflow-deploy.yaml` |
| `examples/helm-paas/` | helm | Chart.yaml, values.yaml |
| `examples/scoredev-paas/` | score | score.yaml |
| `examples/springboot-paas/` | springboot | application.yaml |
| `examples/ably-config/` | ably | ably.json |
| `examples/backstage-idp/` | backstage | catalog-info.yaml |
| `examples/ops-workflow/` | opsworkflow | ops-workflow.yaml |

### Documentation

| Path | Purpose |
|---|---|
| `docs/agentic-gitops/03-worked-examples/` | 3 full DRY→WET worked examples + 8-card story matrix |
| `docs/agentic-gitops/04-schemas/` | JSON schemas for generator-contract, provenance, inverse-transform-plan |
| `examples/demo/simulate-repo-wizard.sh` | 5-step wizard simulation (discover→import→publish→verify→attest) |

### Golden file for c3agent triple (reference)

`cmd/cub-gen/testdata/parity/gitops-import-c3agent.golden.json` — 276 lines showing the full ImportFlowResult with contracts, provenance, field_origin_map, inverse_edit_pointers, inverse_transform_plans.

---

## CI / branch protection

- Main is protected: requires PR with 2 passing checks (Build + Unit, CLI Parity)
- `make ci` runs `gofmt` check + `go test ./...`
- Golden files regenerated with `UPDATE_GOLDEN=1 go test ./cmd/cub-gen -count=1`
- Use feature branches and `gh pr create` for all changes

---

## Quick commands to verify current state

```bash
cd /Users/alexis/confighub/cub-gen

# Build
go build -o ./cub-gen ./cmd/cub-gen

# See all generators (table)
./cub-gen generators

# See all generator triples as JSON details
./cub-gen generators --json --details | jq '.families[] | {kind, profile, policies}'

# See all generator triples as Markdown details
./cub-gen generators --markdown --details

# See c3agent triple from import flow
./cub-gen gitops import --space platform --json ./examples/c3agent ./examples/c3agent | jq '{generator_kind: .discovered[0].generator_kind, contracts: .contracts[0], inverse_transform_plans: .inverse_transform_plans[0]}'

# Run all demos
./examples/demo/ai-work-platform/run-all.sh

# Full CI
make ci
```
