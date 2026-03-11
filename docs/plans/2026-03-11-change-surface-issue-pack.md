# Change Surface v0.2 - Ready-to-File Issue Pack

Status: Draft issue pack
Owner persona: successful OSS project owner (developer-first adoption, clear value, low-friction onboarding)

Execution/status companion:
- `docs/plans/2026-03-11-action-plan-v01-execution-checklist.md`

## Product posture (applies to every issue)

- Lead with developer outcomes, not framework jargon.
- Time-to-first-value target: <= 10 minutes from clone to actionable output.
- Every new surface must work with existing examples and connected ConfigHub mode.
- Every "new" abstraction must be additive over existing import/publish/verify/attest flow.

## Segment priority (this iteration)

1. (b) Brownfield Spring Boot/Helm users
2. (c) AI + developer collaborative workflows
3. (a) Greenfield platform builders
4. (d) AI-only pilot (guarded)

---

## Epic 1 - Language and adoption surface

### Issue 1: docs(ux): replace "fastpath" with `change preview|run|explain` vocabulary
Labels: `kind/docs`, `area/ux`, `priority/p0`

Problem
- "fastpath" is internal language and does not communicate stable user intent.

Outcome
- Public docs consistently use `change` verbs that map to user jobs-to-be-done.

Scope
- Replace top-level references in README, demo docs, and workflow docs.
- Add short command glossary at top-level README.

Acceptance criteria
- No "fastpath" in top-level README or examples/demo README.
- README contains canonical definitions for `change preview`, `change run`, `change explain`.
- Existing local/connected quickstarts remain runnable.

PR notes
- Keep backward compatibility for existing script names in this issue (rename in later issue if needed).

---

### Issue 2: docs(segmentation): add ranked persona-to-feature matrix with primary-owner mapping
Labels: `kind/docs`, `area/product`, `priority/p0`

Problem
- Feature work is not always tied to one primary segment, weakening adoption clarity.

Outcome
- Every feature maps to one primary audience and one measurable adoption result.

Scope
- Extend `docs/workflows/user-story-acceptance.md` with a ranked segment table.
- Include "primary segment", "secondary segment", "proof artifact" columns.

Acceptance criteria
- Each planned feature has exactly one primary segment.
- Segment ordering matches v0.2 priority.
- Story matrix links to concrete demo scripts/artifacts.

---

## Epic 2 - Compelling "prompt as DRY" narrative

### Issue 3: docs(worked-example): publish canonical `prompt as DRY` story (`prompt-as-dry.md`)
Labels: `kind/docs`, `area/workflows`, `priority/p0`

Problem
- Users ask what "prompt as DRY" means in concrete operational terms.

Outcome
- One canonical example shows natural-language intent compiled into governed artifacts with inverse edit trace.

Scope
- New doc: `docs/workflows/prompt-as-dry.md`.
- Include one Swamp-style English intent ("do this, then that") and one brownfield mapping note (Helm/Spring).
- Show `intent -> generated artifacts -> decision state -> inverse edit guidance -> evidence`.

Acceptance criteria
- Doc includes copy/paste commands and expected JSON snippets.
- Demonstrates same `change_id` across preview/run/explain outputs (or current equivalents until commands land).
- Links from README "Which story should you read first?" section.

---

### Issue 4: examples(demo): add one runnable prompt-first demo in local and connected modes
Labels: `kind/examples`, `area/demo`, `priority/p1`

Problem
- Story docs without runnable proof reduce trust for developers.

Outcome
- Developers can run one prompt-style demo end-to-end in <= 10 minutes.

Scope
- Add demo script pair under `examples/demo/` (local + connected).
- Reuse existing pipeline and emit mutation card + evidence summary.

Acceptance criteria
- Local demo runs without auth and outputs actionable edit guidance.
- Connected demo starts with `cub auth login` and outputs decision/evidence files.
- Included in demo README index.

---

## Epic 3 - First-class CLI surface

### Issue 5: cli(contract): define `cub-gen change preview|run|explain` contract (flags, JSON, exit codes)
Labels: `kind/spec`, `area/cli`, `priority/p0`

Problem
- Current command composition is powerful but fragmented for new users.

Outcome
- One stable, developer-facing CLI contract for humans, CI, and agents.

Scope
- New contract doc under `docs/contracts/`.
- Specify required/optional flags, output schemas, and exit code matrix.

Acceptance criteria
- Contract includes deterministic JSON examples for all three commands.
- Exit codes documented for policy BLOCK/ESCALATE and infra errors.
- Reviewed against existing command semantics for no behavioral contradiction.

---

### Issue 6: cli(mvp): implement `cub-gen change preview` as thin wrapper over import/publish/verify/attest
Labels: `kind/feature`, `area/cli`, `priority/p0`

Problem
- New users need one command for immediate value before full workflow complexity.

Outcome
- `change preview` returns mutation card + evidence pointers in one shot.

Scope
- Implement new subcommand with stable JSON output.
- Reuse existing internals; no logic fork.

Acceptance criteria
- Output includes: `change_id`, `bundle_digest`, top edit recommendation, ownership, confidence.
- Existing tests remain green; add golden tests for new command.
- Works for at least Helm and Spring examples.

---

### Issue 7: cli(mvp): implement `cub-gen change run` with local and connected modes
Labels: `kind/feature`, `area/cli`, `priority/p0`

Problem
- Users need one command to execute governance path consistently in terminal and CI.

Outcome
- `change run` executes preview + (optional connected ingest/query) with one contract.

Scope
- Local mode: no login, local artifacts.
- Connected mode: `cub auth login` path + backend decision query.

Acceptance criteria
- `change run --mode local` and `--mode connected` both supported.
- Connected mode fails fast if unauthenticated or missing context.
- Emitted summary includes source of decision authority.

---

### Issue 8: cli(mvp): implement `cub-gen change explain` for inverse-edit and provenance drilldown
Labels: `kind/feature`, `area/cli`, `priority/p1`

Problem
- Developers debugging live fields need fast source-path explanation.

Outcome
- `change explain` provides direct "what to edit and why" output for a selected field/resource.

Scope
- Support explain by `change_id` + filter path/resource.
- Return owner, source file/path, confidence, edit hint.

Acceptance criteria
- Explain output is consumable by terminal and CI comment bots.
- At least one helm and one spring example included in tests.

---

## Epic 4 - First-class API surface

### Issue 9: api(contract): define `POST/GET /v1/changes*` schemas and examples
Labels: `kind/spec`, `area/api`, `priority/p1`

Problem
- CI/agents/UI adoption needs stable HTTP/JSON contract without shell parsing.

Outcome
- Explicit API contract for change creation, status, and explanations.

Scope
- New API contract doc + JSON schemas in docs/contracts.
- Include auth/error semantics and status transitions.

Acceptance criteria
- Schemas cover decision, mutation card, evidence bundle references.
- Examples map 1:1 to CLI contract fields.

---

### Issue 10: api(mvp): expose compatibility API adapter using existing connected pipeline
Labels: `kind/feature`, `area/api`, `priority/p2`

Problem
- Contract docs without implementation create adoption lag.

Outcome
- Thin adapter endpoint(s) available for early integration tests.

Scope
- Implement minimal server/adaptor path (or documented compatibility layer) reusing existing flow.

Acceptance criteria
- One CI sample consumes API response directly (no jq shell parsing across multiple files).
- Response includes same `change_id` and evidence links as CLI.

---

## Epic 5 - Proof in real invocation contexts

### Issue 11: examples(proof): add terminal proof for `change` surface (brownfield-first)
Labels: `kind/examples`, `area/demo`, `priority/p0`

Problem
- Need immediate proof for core OSS users (Helm/Spring brownfield teams).

Outcome
- One terminal walkthrough shows end-to-end value in <= 10 min.

Scope
- Add script + README section tied to Helm/Spring example.

Acceptance criteria
- Produces mutation card and inverse edit recommendation.
- Includes local and connected variants.

---

### Issue 12: ci(proof): add CI workflow using `change run` non-interactive auth
Labels: `kind/ci`, `area/adoption`, `priority/p0`

Problem
- CI is core enterprise adoption path; should be first-class.

Outcome
- One GitHub Actions workflow demonstrates direct `change run` integration.

Scope
- Add/update workflow with strict connected gates and story evidence checks.

Acceptance criteria
- Non-interactive auth path documented and validated.
- Workflow output publishes summary artifact with `change_id` and decision.

---

### Issue 13: agent(proof): add agent/tool-call path using same `change_id` lifecycle
Labels: `kind/examples`, `area/ai`, `priority/p1`

Problem
- AI+dev segment needs proof that agent and human paths are unified.

Outcome
- One agent invocation path shows same contract/evidence lifecycle.

Scope
- Add demo script and sample tool-call payload/response.

Acceptance criteria
- Output reuses CLI/API contract fields exactly.
- Evidence chain includes actor identity in unified format.

---

## Epic 6 - AI-only guardrails

### Issue 14: policy(safety): define AI-only allowed-scope matrix + mandatory rollback hooks
Labels: `kind/policy`, `area/safety`, `priority/p1`

Problem
- AI-only mode has highest risk and must not bypass governance.

Outcome
- Explicit guardrail policy for what AI-only can change and how rollback is enforced.

Scope
- New policy doc under `docs/workflows/`.
- Define prohibited scopes, required approvals, rollback trigger rules.

Acceptance criteria
- AI-only path is explicitly blocked for out-of-scope mutations.
- Rollback proposal path documented and tied to evidence artifacts.

---

## Filing sequence (recommended)

1. #1 language freeze
2. #2 segment matrix
3. #3 prompt-as-dry doc
4. #5 CLI contract
5. #6 preview MVP
6. #7 run MVP
7. #8 explain MVP
8. #11 terminal proof
9. #12 CI proof
10. #13 agent proof
11. #9 API contract
12. #10 API adapter MVP
13. #14 AI-only safety matrix
14. #4 prompt demo script polish (can be parallel with #11)

## Definition of success for this pack

- A new developer can run one command and understand: what changed, what to edit, why, and whether it is safe.
- A platform owner can adopt connected mode with a documented CI checklist and deterministic outputs.
- A contributor can read issues and execute in small PR slices without reinterpreting product intent.
