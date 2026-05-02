# Decision + Attestation State Contract

This contract defines how bridge artifacts move from ingest to governed decision.

## Scope

1. `cub-gen` publishes deterministic change bundles and attestations.
2. ConfigHub stores governed WET state and decision authority.
3. Bridge decision records link these by shared `change_id` and digest evidence.

## State model

Decision state is explicit and finite:

1. `INGESTED`: bundle accepted by governed WET ingest.
2. `ATTESTED`: verified attestation linked to bundle digest.
3. `ALLOW`: explicit decision to proceed.
4. `ESCALATE`: explicit decision to require additional approval.
5. `BLOCK`: explicit decision to stop.

Terminal states are `ALLOW | ESCALATE | BLOCK`.

## Required identity links

Every decision record (`cub.confighub.io/governed-decision-state/v1`) must carry:

1. `change_id`
2. `trace_id` (normally the same value as `change_id`)
3. `bundle_digest`
4. `state`
5. `updated_at`
6. `proof_events[]`

If state is `ATTESTED` or terminal, it must also carry:

1. `attestation_digest` (digest-linked evidence)

The bundle, attestation, and governed decision artifacts carry
`proof_events[]`. These are log-safe records for Pilot and validation flows:

1. bundle event: `change_bundle.published`
2. attestation event: `attestation.verified`
3. decision events: `governed_decision.created`,
   `governed_decision.attested`, and `governed_decision.applied`
4. shared join fields: `trace_id`, `change_id`, artifact digest, parent
   artifact digest, parent event id, and decision state

If state is terminal, it must also carry:

1. `decision_reason`
2. `decided_at`
3. one explicit authority: `approved_by` or `policy_decision_ref`

This enforces the invariant: no implicit deploy.

## Transition rules

Allowed transitions:

1. `INGESTED -> ATTESTED`
2. `ATTESTED -> ALLOW`
3. `ATTESTED -> ESCALATE`
4. `ATTESTED -> BLOCK`

Disallowed:

1. Any terminal decision without attestation linkage.
2. Any terminal decision without explicit authority.
3. Any direct `INGESTED -> ALLOW|ESCALATE|BLOCK`.
4. Any locally produced decision proof event that cannot be joined by
   `trace_id` and `change_id`.

## Query by `change_id`

Bridge query path:

1. `GET /api/v1/governed-wet-decisions/{change_id}`

The response is validated against the decision-state contract and must return the same `change_id` that was requested.

## Implementation anchors

1. Contract and transition enforcement: `internal/bridge/decision.go`
2. End-to-end state/query tests: `internal/bridge/decision_test.go`
3. Proof-event schema and validation: `internal/proof/event.go`, `internal/contracts/schemas/proof-event.v1.schema.json`
