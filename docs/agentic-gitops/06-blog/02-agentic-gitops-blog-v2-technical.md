# Agentic GitOps: A Technical Operating Model for AI-Scale Change

GitOps solved a major operations problem: make desired state explicit and continuously reconcile runtime to it.

In enterprise environments, that gave teams reproducibility, rollback confidence, and a clear deployment contract. But AI introduces a new scale problem: mutation volume and mutation autonomy increase faster than human review capacity.

The question is no longer just “can we converge?” It is also:
- Was this mutation authorized?
- Under what policy and risk tier?
- Did execution stay within scoped authority?
- Did observed outcome match intended and approved outcome?
- Is there an attested proof chain for audit and incident response?

This is where **agentic GitOps** starts: not as a replacement for Flux/Argo, but as a governance extension around the reconciliation loop.

Qualification rule:
If the active inner reconciliation loop (`WET -> LIVE`) is absent, classify the flow as `governed config automation` rather than Agentic GitOps.

## 1. What classical GitOps does well

Classical GitOps establishes:
- declarative desired state
- versioned change history
- pull-based reconciliation
- runtime convergence

These are necessary and remain foundational.

## 2. What classical GitOps does not fully solve at AI scale

When AI systems propose or execute many changes per day, teams hit four recurring gaps.

### Gap A: Intent is underspecified
A diff shows what changed, not why it was proposed, what evidence triggered it, or what risk objective it served.

### Gap B: Authority is ambiguous
A merge is not a risk decision. In enterprise settings, execution rights should depend on policy tier, domain, and approval conditions.

### Gap C: Verification is fragmented
Outcome signals are split across CI, reconciler events, monitoring, and tickets. This impairs confidence and slows incident response.

### Gap D: Reverse path is weak
Urgent runtime interventions can be reverted by reconciliation unless there is an explicit governed write-back path.

## 3. Core claim

Agentic GitOps extends GitOps from a **convergence contract** to a **governed mutation contract**.

Convergence remains the inner loop.
Governance becomes the outer loop.

## 4. Two-loop architecture

### Inner loop: reconciliation
- Flux/Argo continuously reconcile intended state to runtime.
- Drift is detected and corrected according to declared state.

### Outer loop: mutation governance
- Human and/or AI proposes mutation intent.
- Policy evaluates trust tier and constraints.
- Decision is issued: `ALLOW`, `ESCALATE`, or `BLOCK`.
- Scoped execution authority is issued only on `ALLOW` (or after approved escalation).
- Outcome is verified against approved intent/spec.
- Attestation closes the mutation chain.

Lifecycle:

`propose -> evaluate -> approve/escalate/block -> execute -> verify -> attest`

## 5. Non-negotiable control rules

For AI-enabled operations, three controls should be hard requirements.

1. **No execution authority without policy decision.**
2. **No success claim without verification evidence.**
3. **No closed mutation record without attestation.**

These are not reporting improvements. They are control-plane requirements.

## 6. Bidirectional GitOps without implicit overwrite

“Bidirectional” often causes confusion. The correct model is:
- runtime observations can generate proposals back to intended state
- proposals are reviewed/evaluated in the same governance path
- nothing in runtime silently mutates desired state

This preserves GitOps discipline while enabling safe reverse flow.

## 7. Data model: governance-grade mutation records

A practical unit is a mutation card (for example, `ChangeInteractionCard`) linked to a commit.

Typical fields:
- identity: repo, commit, actor, agent
- intent: what was requested and why
- evidence: signals that triggered proposal
- decision: policy refs, trust tier, `ALLOW|ESCALATE|BLOCK`
- execution: runtime, scope, token context
- verification: expected vs observed checks
- outcome: applied/failed/rolled back, health summary
- attestation: signer chain and timestamps

Git can store compact linkage artifacts (trailers + receipts). A control plane can store richer queryable state.

## 8. Tool boundary model

A clean boundary keeps adoption practical.

- **Flux/Argo:** reconciliation runtime.
- **Git:** primary collaboration and ingress surface for most teams; immutable linkage artifacts.
- **Mutation ledger (e.g., cub-track):** commit-linked mutation history.
- **Observer (e.g., cub-scout):** runtime evidence and drift context.
- **Control plane (e.g., ConfigHub):** stores dry units and wet units (WET is the deployment contract), plus policy decisioning, provenance graph, verification state, attestation indexing, and cross-repo queries.

This enables OSS-first adoption with optional deeper governance integration.

## 9. Start-today adoption path for Flux/Argo teams

This is the critical adoption strategy: no migration and no rip-and-replace.

### Step 1: keep reconciler unchanged
Keep Flux/Argo exactly as deployed.

### Step 2: add mutation ledger in Git
Capture per-change intent/decision/outcome links in the repo.

### Step 3: connect ConfigHub as control plane
Store dry units and wet units in ConfigHub, with WET authoritative for deployment.
Keep Git as collaboration/ingress for most teams.

### Step 4: add decision states
Adopt explicit policy outcomes: `ALLOW`, `ESCALATE`, `BLOCK`.

### Step 5: enforce verification for production mutations
Require verification artifacts tied to the mutation record.

### Step 6: add attestation chain
Persist who approved, what executed, and observed outcome for audit and incident review.

### Step 7: scale governance
Add cross-repo queryability, trust-tier enforcement, scoped execution tokens, and compliance export.

## 10. Trust-tier execution model

A tiered model controls blast radius:
- Tier 0: observe only
- Tier 1: low-risk domains with auto-allow policy
- Tier 2: medium-risk requires escalation/human approval
- Tier 3: high-risk/prod requires strong attestation and dual approval

AI systems can propose across tiers, but execution rights are tier-gated.

## 11. Example mutation flow (production scaling request)

Scenario: AI proposes increasing replicas from 4 to 10 for a production service.

1. Proposal submitted with supporting evidence.
2. Policy classifies request as Tier 2 and returns `ESCALATE`.
3. Human reviewer approves constrained execution.
4. Reconciler applies updated intended state.
5. Verification checks rollout health and SLO impact.
6. Attestation is written and linked to mutation record.

Result: operations can move quickly without losing authority, traceability, or proof.

## 12. Anti-patterns to avoid

- Treating AI output as trusted by default.
- Using “merged to main” as the only approval signal.
- Equating reconciliation success with business/safety success.
- Allowing runtime fixes to bypass governed reverse flow.
- Storing evidence in unstructured chat/log fragments only.

## 13. Success metrics

Measure progress with operational and governance indicators:
- policy decision latency (proposal to `ALLOW/ESCALATE/BLOCK`)
- verification completeness rate for production mutations
- attestation completeness rate
- incident MTTR for config-induced issues
- percentage of mutations with full intent->decision->outcome chain

## 14. What “post-Flux/post-Argo” really means

It does not mean replacing Flux or Argo.
It means adding a governance/control layer suitable for AI-era mutation rates.

Flux/Argo remain the convergence engine.
Agentic GitOps adds governed mutation authority, verification, and attestation.

That combination is what makes platform operations both faster and safer as AI participation increases.
