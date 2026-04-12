# Remote Stores And Projection Targets

This document is the written model for `#221`.

The short version:

1. many systems can produce or own operational config
2. ConfigHub is the governed operational authority layer across them
3. some approved values should route back out to downstream systems explicitly

## The model

A remote system may play one or more roles:

1. **authoring source**
   where compact human intent starts
2. **upstream producer**
   a system that currently emits the config we ingest
3. **narrow authority**
   a system that remains authoritative for a defined field/object scope
4. **governed operational authority**
   the normalized, reviewable ConfigHub state we mutate and approve
5. **downstream projection target**
   a system that should receive approved fresh config after policy checks
6. **evidence source**
   a system we observe for rendered/live proof

Git is only one member of that set.

## Why this matters to cub-gen

`cub-gen` already has the right primitives:

1. `GeneratorContract` tells us what deterministic generator boundary ran
2. `ProvenanceRecord` tells us where fields and outputs came from
3. `InverseTransformPlan` tells us where a safe upstream edit should go
4. route language already distinguishes "apply here", "lift upstream", and
   "block/escalate"

The missing product language was how to talk about remote systems that are not
just files in Git.

## Decided route language

`project downstream` should be treated as a first-class route beside:

1. `apply here`
2. `lift upstream`
3. `block/escalate`
4. `project downstream`

Meaning:

1. the governed value is approved in ConfigHub
2. publication to the downstream remote store is explicit
3. publication is policy-checked and auditable
4. publication does not erase the upstream provenance chain

## Minimum metadata we need

For any remote store / external authority / projection target, the minimum model
needs to capture:

1. system identity
2. system type
   Examples: `git`, `helm`, `vault`, `aws`, `gcp`, `saas`, `custom-api`
3. authority scope
   Which fields or object domains it actually owns
4. import mode
   Snapshot, pull, event-driven, generated, observed-only
5. publish mode
   Manual, policy-gated push, controller-managed, unsupported
6. freshness reference
   Revision, resource version, watermark, timestamp, or equivalent
7. publish receipts
   What was written, where, and under which approved decision
8. loop-prevention linkage
   Enough identity to detect "same system came back around as fresh input"

## How this interacts with the contract triple

### GeneratorContract

`GeneratorContract` remains the deterministic DRY -> WET contract.

Remote-store implications:

1. a contract may declare that some upstream authority is outside Git
2. a contract may state whether downstream projection is allowed for a given
   family
3. deterministic local generation and remote projection remain separate phases

### ProvenanceRecord

`ProvenanceRecord` is where remote-store reality should accumulate.

It should carry:

1. imported source references from remote producers
2. authority annotations for fields or objects
3. freshness/watermark evidence
4. downstream publish receipts
5. projection provenance linking approved state to actual remote publication

### InverseTransformPlan

`InverseTransformPlan` remains the safe-edit plan.

With remote stores in the model, a plan can now route to:

1. local governed mutation
2. lift upstream to Git or another producer
3. block/escalate
4. project downstream to an allowed target

## Freshness and loop prevention

If the same remote system can both feed ConfigHub and receive updates back from
ConfigHub, we need explicit loop controls:

1. store the imported revision/watermark
2. store the publish receipt for the downstream write
3. compare fresh imports against the last published revision
4. suppress or flag loops when the same publication echoes back as a new import
5. escalate when a stale import collides with a fresher approved publish

## Concrete example path in this repo

The best current in-repo example path is the Spring platform story:

1. source stays partly in Git
2. operational state is governed locally/through ConfigHub concepts
3. route language already distinguishes app-owned, upstream-owned, and
   platform-owned fields

That makes Spring the right first candidate for an eventual mocked non-Git
remote store example where an approved value is explicitly `project downstream`.

## Scope call for this issue

This issue is satisfied by:

1. a written conceptual model
2. an explicit decision that `project downstream` is first-class route language
3. documented interaction with contracts, provenance, inverse plans, and route
   semantics
4. minimum metadata for freshness, publish receipts, and loop prevention

Implementation of real remote-store projection flows can stay incremental after
the model is agreed.
