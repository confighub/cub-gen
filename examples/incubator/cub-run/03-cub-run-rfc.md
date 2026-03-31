# RFC: `cub run` and `Operation` records

## Summary

ConfigHub already stores desired state.

For connected procedures, it should also store structured operational state.

This RFC proposes two things:

- a first-class `Operation` record in ConfigHub for bounded procedures
- `cub run` as the OSS CLI over that record

This is not mainly about a new command.
It is about making multi-step operations first-class in ConfigHub instead of leaving them split across shell output, GUI pages, worker activity, and user memory.

## Problem

Some important ConfigHub procedures are not one command.

Examples:

- publish to ArgoCD, then wait for reconcile, then check health
- resolve worker and target, then apply, then verify readiness
- push an upgrade through layered units, then confirm the resulting state

Today, users can often start these procedures, but after that they still have to reconstruct:

- what step was reached
- what system is now responsible
- whether the procedure is still running or merely waiting
- which checks have actually passed
- whether the visible state is fresh or stale

This slows users down in concrete ways:

- they do not know what to do next
- they rerun work unnecessarily
- they check multiple systems for status
- they reconstruct interrupted work from terminal history

## Why existing ConfigHub support is not enough

ConfigHub could already store operational data in principle.

What is missing is a standard bounded-procedure record:

- one standard shape
- one standard lifecycle
- one standard way to create, update, and read it

Without that, ConfigHub can hold fragments of operational evidence, but each flow still invents its own way of producing and interpreting those fragments.

## Proposal

Add a first-class `Operation` record in ConfigHub.

`cub run` is the CLI for creating, updating, and reading `Operation` records.

## Canonical model

### `Operation`

An `Operation` is the structured record of one bounded procedure.

Minimum fields:

- `OperationID`
- `Procedure`
- `ProcedureVersion`
- `State`
- `SubjectBinding`
- `ApplyMode`
  - `direct`
  - `argo`
  - `flux`
- `ResolvedBindings`
  - worker
  - target
  - controller when delegated
- `Steps`
- `Assertions`
- `ExternalRefs`
  - bundle or publish refs
  - delegated controller refs when relevant
- `Timestamps`
- `EvidenceRefs`
- `Actor`

### Lifecycle

Minimum operation states:

- `running`
- `waiting`
- `done`
- `failed`

Minimum assertion states:

- `pending`
- `pass`
- `warn`
- `fail`

### Key distinction

- `done` means the procedure reached its current endpoint
- `asserted` means the intended state was actually checked

Example:

- publish to ArgoCD can be `done`
- workload health can still be `pending`

That distinction is the main reason this should exist as more than shell output.

### Apply mode should be explicit

Execution behavior should not be inferred indirectly from subject type.

If an example, bundle, or procedure needs to describe how state reaches the cluster, prefer explicit metadata such as:

```yaml
apply: direct
```

or:

```yaml
apply: argo
```

or:

```yaml
apply: flux
```

`kind` may still be useful as a subject classification, but `apply` is what determines completion semantics, waiting behavior, and default assertions.

## CLI surface

Initial CLI shape:

```bash
cub run <procedure> [subject]
cub run get <operation-id>
cub run list
cub run watch <operation-id>
```

Initial procedures:

```bash
cub run global-app/install
cub run deploy <space/unit>
cub run publish <space/unit>
cub run upgrade <space/unit>
cub run validate <space/unit>
```

## Concrete failure modes without this

### 1. Delegated GitOps stays ambiguous

Current reality:

1. publish desired state
2. ArgoCD reconciles later
3. health checks happen after that

Without an `Operation` record, users see command completion but still have to infer what is outstanding.

With an `Operation` record, ConfigHub can show:

- publish step done
- reconcile step waiting or done
- health assertions pending, pass, or fail

### 2. Worker setup failures are invisible

The `gitops-import` example requires four manual steps: create-cluster → install-argocd → setup-apps → install-worker. During e2e testing, the worker pod started successfully, authenticated, and connected to the ConfigHub event stream (HTTP 200). But target registration silently failed because the org was at target quota.

From the user's perspective: the install script printed success, the pod was running, but `cub worker list` showed "Disconnected" with no explanation. Diagnosing this required reading pod logs and cross-referencing quota limits across multiple systems.

With an `Operation` record, this would be:

```text
Step 4/4: install worker     DONE
Assertion: worker connected  FAIL — exceeded target quota
Assertion: targets registered PENDING
```

The failure is visible immediately instead of hidden behind a successful pod startup.

### 3. Live `global-app` install stays hard to run and review

Current reality:

1. create base spaces and base units
2. create environment spaces and clone/mutate units
3. resolve target and apply infra before app units

Without an `Operation` record, the user has commands and scripts but no single operational record of the install procedure.

With an `Operation` record, ConfigHub can show:

- whether preflight passed
- whether config materialization finished
- whether infra and app apply ran in the right order
- which assertions passed afterwards

## MVP

For MVP:

- procedure profiles are hardcoded in `cub`
- ordered steps are hardcoded in `cub`
- default assertions are hardcoded in `cub`
- only `cub run` emits `Operation` records
- existing commands such as `cub unit apply` and `cub function do` stay unchanged

The first concrete procedure profile should be `global-app/install`.

Its top-level phases should be:

- `preflight`
- `materialize-config`
- `apply`
- `assert`

Verbose mode may show substeps inside those phases, but the top-level procedure should stay that simple.

For connected mutating procedures:

- `Operation` records should persist by default
- local-only ephemeral mode should be explicit opt-out

## `watch` and re-derivation

MVP does not require a long-lived server-side run engine.

Instead:

- `cub run` executes locally in the CLI
- `get` reads stored `Operation` state from ConfigHub
- `watch` re-derives current state by polling underlying systems using the stored `Operation` plus the hardcoded procedure profile

To make that work, the stored `Operation` must include enough context for re-derivation, at minimum:

- procedure
- subject binding
- apply mode
- resolved worker and target
- delegated controller ref when relevant
- assertion set or profile to re-check
- bundle or publish reference when needed
- any delegated controller reference needed for status lookup

`cub run list` should show persisted `Operation` records only.

## `--assert`

For MVP, `--assert` should mean:

- evaluate assertions when evidence is available
- print assertion outcomes
- return non-zero if any evaluated required assertion fails

Notes:

- `warn` does not fail the command
- `pending` does not fail the command by itself
- a delegated flow may return `waiting` with pending assertions

## Open questions

- What is the right ConfigHub backing type for `Operation` records?
- What is the minimum assertion set for `global-app/install`, `deploy`, and `publish`?
- When should `cub run` stop waiting automatically versus return `waiting`?
- When, if ever, should procedure profiles become declarative rather than hardcoded?
