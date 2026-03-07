# Agentic GitOps: From Reconciliation to Governed Execution

Qualification rule:
Use `Agentic GitOps` only when an active inner reconciliation loop (`WET -> LIVE`) exists via Flux/Argo (or equivalent reconciler). Without that loop, classify the flow as `governed config automation`.

GitOps gave us a major operational upgrade: desired state is versioned, reconcilers pull it, and runtime converges.

That model works. But AI changes the scale and speed of change so much that convergence alone is no longer enough.

If an AI system proposes dozens of operational mutations per day, teams need to answer more than:
- What is deployed?
- Is it converged?

They also need:
- Why was this change proposed?
- Who authorized it?
- What checks ran before execution?
- Was the real outcome verified afterward?
- Where is the attested proof chain?

That is the shift from classical GitOps to **agentic GitOps**.

## First, clear terms

To avoid confusion, we should separate two different kinds of automation:
- **Reconcilers**: Flux and Argo CD continuously reconcile desired state to runtime state.
- **AI agents**: systems that propose or execute operational/config changes based on prompts, signals, or policy goals.

Reconcilers are excellent at convergence. They are not, by themselves, a governance system for AI-authored mutation intent.

## Why classical GitOps is necessary but insufficient

Classical GitOps gives deterministic deployment flow and rollback mechanics. But enterprise teams face harder questions under AI-driven mutation volume:

1. **Mutation intent is weakly captured**
A Git diff tells you what changed, not why it was proposed, what evidence triggered it, or what risk classification was applied.

2. **Authority is implicit**
A merge may pass CI and still be unacceptable for a given risk tier or production domain unless explicit policy and approval boundaries are enforced.

3. **Outcome proof is fragmented**
Evidence often ends up scattered across PRs, CI logs, reconciler events, chat threads, and ticket comments.

4. **Break-glass and live changes are awkward**
Strict one-way interpretation of GitOps can revert urgent live interventions unless there is a governed reverse path back to intended state.

## The model: two loops, one governance chain

A practical model is:

1. **Outer loop (intent/governance)**
   - Human and/or AI proposes a mutation.
   - Policy evaluates risk and trust tier.
   - Decision is issued: `ALLOW`, `ESCALATE`, or `BLOCK`.
   - If allowed, scoped execution authority is issued.

2. **Inner loop (reconciliation)**
   - Flux/Argo reconcile intended state to runtime.
   - Runtime outcomes are observed.

3. **Verification + attestation closes the chain**
   - Verification checks whether observed outcome matches approved intent/spec.
   - Attestation records who authorized, what ran, and what happened.

This gives a complete mutation lifecycle:

`propose -> evaluate -> approve/escalate/block -> execute -> verify -> attest`

## Hard rule for agentic operations

In an enterprise setting, this must be explicit and enforceable:

- **No execution authority without policy decision**.
- **No success claim without verification evidence**.
- **No closed mutation record without attestation**.

In short: governance is not a report you write later; it is a control surface applied before and after execution.

## Where bidirectional GitOps fits

“Bidirectional” does **not** mean runtime silently overwrites desired state.

It means:
- Runtime observations can produce governed proposals to intended state.
- Those proposals pass through policy, approval, verification, and attestation.
- Nothing is implicitly promoted.

That keeps your original GitOps discipline intact while making live operations survivable under AI-era mutation rates.

## Start today if you already use Flux or Argo

This is the critical adoption point: **you do not need to replace Flux/Argo**.

You can add this model incrementally, today:

1. **Keep your reconciler exactly as-is**
   Flux/Argo remains your runtime convergence engine.

2. **Add a Git-native mutation ledger in the repo**
   Install `cub-track` (OSS) and enable commit-linked mutation records.
   You get immediate “why/decision/outcome” traceability per change.

3. **Connect ConfigHub as the control-plane store**
   Store dry units and wet units in ConfigHub, with WET authoritative for deployment.
   Keep Git as the primary collaboration and ingress surface for most teams.

4. **Introduce lightweight governance gates**
   Start with explicit decision states (`ALLOW | ESCALATE | BLOCK`) and human approval for medium/high-risk domains.

5. **Require verification artifacts for production mutations**
   Tie rollout result checks to the mutation record.

6. **Add attestation for audit-grade chain of trust**
   Persist who approved, what was executed, and observed outcome, linked to the commit and mutation card.

This gives meaningful value at each step, without platform migration.

## Tooling model (plain English)

A simple boundary model:

- **Flux/Argo**: reconcile runtime.
- **Git**: primary collaboration and ingress surface for most teams; immutable linkage artifacts.
- **Mutation ledger (`cub-track`)**: structured mutation history tied to commits.
- **Observer (`cub-scout`)**: evidence about runtime reality and drift.
- **ConfigHub**: control-plane store for dry units and wet units, with WET authoritative for deployment; centralized policy decisioning, provenance, verification state, attestation indexing, and cross-repo queryability.

You can run OSS-first and connect deeper governance later.

## Why this is “post-Flux/post-Argo” without replacing either

“Post-Flux/post-Argo” means the next layer of capability, not a replacement:

- Flux/Argo solve convergence.
- Agentic GitOps adds governed mutation intent, verification, and attestation.
- The combined system supports both app GitOps and AI GitOps at enterprise scale.

The reconciler remains. The control plane around mutation authority becomes modern.

## What this looks like in practice

Imagine an AI agent proposes scaling a production service from 4 to 10 replicas due to forecasted load.

A governed path should look like this:

1. Proposal captured with intent + evidence.
2. Policy classifies as Tier 2 (medium/high impact) and returns `ESCALATE`.
3. Human approver reviews and authorizes constrained execution.
4. Reconciler applies desired state.
5. Verification checks rollout health and SLO guardrails.
6. Attestation is recorded and linked to the commit/mutation record.

Later, incident review or audit is a query, not archaeology.

## What changes for teams

- Platform teams gain explicit authority boundaries for AI.
- Security/compliance teams gain proof chains they can trust.
- App teams keep familiar Flux/Argo workflows.
- Leadership gets faster change velocity without giving up control.

## The near-term direction

Adoption path:
- **Now**: anyone using Flux/Argo can add mutation-ledger governance incrementally and connect ConfigHub as dry+wet control plane.
- **Soon**: an agentic-only operating mode where governed intent/verification/attestation is first-class even outside traditional repo-centered workflows.

The point is not to abandon GitOps.
The point is to make GitOps robust for an era where machines propose and execute far more change than humans can manually reason about.

## Closing

Classical GitOps answered: “Can systems converge to declared intent?”

Agentic GitOps must answer two additional questions:
- “Was this mutation authorized under explicit policy?”
- “Can we prove the real-world outcome matched approved intent?”

If your system cannot answer those two questions quickly and reliably, it is not ready for AI-scale operations.
