# Agentic GitOps (LinkedIn Cut)

Qualification rule:
Use `Agentic GitOps` only when an active inner reconciliation loop (`WET -> LIVE`) exists via Flux/Argo (or equivalent reconciler). Without that loop, classify the flow as `governed config automation`.

AI is not replacing GitOps. It is making GitOps more important.

Classical GitOps gives us a powerful contract:
- declare desired state in Git
- let Flux/Argo reconcile to runtime

But at AI speed, convergence is not enough.

When AI systems propose operational changes at scale, teams need to answer:
- Why was this mutation proposed?
- Who authorized it?
- What checks ran before execution?
- Was the outcome verified?
- Where is the attested proof chain?

That is **agentic GitOps**:

`propose -> evaluate -> approve/escalate/block -> execute -> verify -> attest`

Three hard rules:
1. No execution authority without policy decision.
2. No success claim without verification evidence.
3. No closed mutation record without attestation.

Important nuance: “bidirectional GitOps” does **not** mean runtime silently overwrites desired state.
It means live observations can create governed proposals back to intended state.

## Start today (if you already use Flux or Argo)

You can adopt incrementally without migration:
1. Keep Flux/Argo exactly as-is.
2. Add a Git-native mutation ledger (`cub-track`) for commit-linked why/decision/outcome.
3. Connect ConfigHub as control plane for dry+wet units (WET is the deployment contract).
4. Add decision states (`ALLOW | ESCALATE | BLOCK`) with trust tiers.
5. Require verification + attestation for production mutations.

Near-term path:
- **Now:** anyone on Flux/Argo can add this governance layer.
- **Soon:** agentic-only operating mode with first-class governed execution.

“Post-Flux/post-Argo” is not replacement. It is the next layer: governed mutation authority for the AI era.
