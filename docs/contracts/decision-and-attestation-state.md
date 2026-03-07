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
2. `bundle_digest`
3. `state`
4. `updated_at`

If state is `ATTESTED` or terminal, it must also carry:

1. `attestation_digest` (digest-linked evidence)

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

## Query by `change_id`

Bridge query path:

1. `GET /api/v1/governed-wet-decisions/{change_id}`

The response is validated against the decision-state contract and must return the same `change_id` that was requested.

## Implementation anchors

1. Contract and transition enforcement: `internal/bridge/decision.go`
2. End-to-end state/query tests: `internal/bridge/decision_test.go`
