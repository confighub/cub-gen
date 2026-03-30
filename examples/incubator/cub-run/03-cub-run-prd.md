# PRD: `cub run`

## Status

Draft

## Summary

ConfigHub already supports operations that span more than one low-level CLI action.

ConfigHub should also act as the system of record for bounded operational procedures in connected environments.

This PRD proposes:

- a first-class `Operation` record in ConfigHub
- `cub run` as the public OSS CLI for creating, updating, and reading those records

The point is not mainly a new command.
The point is to stop treating multi-step operations as shell output plus memory.

Repository context:

- worked scenarios and examples live in [`confighub/examples`](https://github.com/confighub/examples)
- the public OSS `cub` CLI lives in [`confighub/sdk/cmd/cub`](https://github.com/confighub/sdk/tree/main/cmd/cub)

## What `cub run` adds

ConfigHub could already store operational data in principle.

What is missing today is a standard bounded-procedure record:

- a standard shape
- a standard lifecycle
- a standard CLI for creating, updating, and reading it

Without `cub run`, ConfigHub can hold fragments of operational evidence, but each flow still invents its own way of producing and interpreting those fragments.

With `cub run`, ConfigHub gets one consistent operational record for a bounded procedure.

## User Problem

Some important ConfigHub tasks are already bounded procedures, not single commands.

Examples:

- preflight plus target selection plus apply
- publish plus delegated ArgoCD or Flux reconcile
- upgrade plus wait plus assertions
- onboarding flows where worker, target, and auth must be checked before continuing

In those cases, users need to know:

1. where am I?
2. am I done?
3. what failed, and what is still waiting?
4. what has actually been proven?

Without that, users end up stitching the procedure together from shell scripts, terminal output, GUI pages, and memory.

More importantly, the system itself has no clear shared record of the operation as it progressed.

That is already visible in our examples.

For example, the current `realistic-app` example in `confighub/examples` uses:

- `setup.sh` at 72 lines
- `set-target.sh` at 25 lines
- `verify.sh` at 117 lines
- `lib.sh` at 453 lines

Not all of that logic should move into `cub`, but the repeated procedure mechanics should: step order, progress reporting, waiting, assertion reporting, and operation recording.

## How this helps users get apps up faster

`cub run` does not make Kubernetes, ArgoCD, or workers execute faster.

It reduces four specific delays:

### 1. Delay from not knowing what to do next

Users lose time when they cannot tell which step they are on or what the next waiting point is.

### 2. Delay from rerunning work unnecessarily

Users lose time when they cannot tell whether preflight, publish, apply, or target resolution already happened.

### 3. Delay from checking multiple systems for status

Users lose time when they have to inspect CLI output, ConfigHub, ArgoCD, and cluster state separately just to know whether the app is actually up.

### 4. Delay from interruption or handoff

Users lose time when a later human or AI has to reconstruct what happened from terminal history instead of reading one operational record.

## Goal

Give ConfigHub one consistent operational record for each connected bounded procedure, and give `cub` one consistent CLI surface over that record.

For a user, success looks like this:

- they can start one named procedure from `cub`
- they can see the current step
- they can see which steps completed, failed, or are still pending
- they can see which assertions passed, failed, warned, or are still pending
- they can tell whether the procedure is done
- the same operational record is visible later in ConfigHub

## Canonical model

The center of gravity should be the stored object, not the CLI verb.

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
  - bundle refs
  - publish refs
  - delegated controller refs
- `Timestamps`
- `EvidenceRefs`
- `Actor`

### `Step`

A concrete stage inside the procedure.

Minimum fields:

- `Name`
- `State`
- `StartedAt`
- `FinishedAt`
- `Message`

### `Assertion`

A named check about current or resulting state.

Minimum fields:

- `Name`
- `State`
- `Required`
- `LastEvaluatedAt`
- `EvidenceRef`
- `Message`

### Separation from desired state

`Operation` records are operational data.

They should not appear as ordinary mutations on app or deployment units.
They belong in ConfigHub, but distinct from desired-state unit data.

### Apply mode should be explicit

Execution semantics should not be inferred indirectly from subject type.

If an example, bundle, or procedure needs to describe how desired state reaches the cluster, prefer:

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

`kind` can still be useful as subject classification, but `apply` is what determines waiting behavior, completion semantics, and default assertions.

## Lifecycle

Suggested operation states:

- `running`
- `waiting`
- `done`
- `failed`

Suggested step states:

- `pending`
- `running`
- `done`
- `failed`
- `skipped`

Suggested assertion states:

- `pending`
- `pass`
- `warn`
- `fail`

## Critical distinction: done vs asserted

This is the core behavioral distinction.

- `done`
  - the procedure reached its current endpoint
- `asserted`
  - the intended state was actually checked

Example:

- publish to ArgoCD can be `done`
- application health can still be `pending`

So `cub run` must show both:

- procedure state
- assertion state

A plain `echo "done"` only covers step completion. It does not tell the user whether the result was proven.

## Procedure profiles

For MVP, procedure profiles should be hardcoded in `cub`.

That means:

- procedure names are hardcoded in the CLI
- ordered steps are hardcoded in the CLI
- default assertions are hardcoded in the CLI
- required inputs and defaults are hardcoded in the CLI

This keeps the first version understandable and shippable.

Later, ConfigHub may choose to externalize procedure profiles into declarative definitions. That is out of scope for MVP.

## Procedure profile contract

Each hardcoded procedure profile should define at least:

- subject resolver
- worker resolver
- target resolver
- apply mode resolver
- step evaluators
- assertion evaluators
- external refs to store for later `watch`

This is what makes `watch` and `get` deterministic instead of ad hoc.

## First concrete procedure profile

The first concrete profile should be `global-app/install`.

Top-level phases:

- `preflight`
- `materialize-config`
- `apply`
- `assert`

What each phase covers:

- `preflight`
  - auth and context
  - prefix reuse policy
  - target presence if live mode is requested
- `materialize-config`
  - create base spaces and units
  - create environment spaces
  - clone, mutate, and link units
- `apply`
  - apply infra first
  - then apply app units when live mode is requested
- `assert`
  - config shape is correct
  - targets are set when expected
  - apply completed when requested
  - health checks passed when evidence is available

Verbose mode may show substeps inside these phases, but the top-level procedure should stay at this level.

## Proposed CLI surface

### Main verb

`cub run` should be the primary CLI verb for bounded procedures.

### MVP commands

```bash
cub run <procedure> [subject]
cub run get <operation-id>
cub run list
cub run watch <operation-id>
```

### Initial procedures

```bash
cub run global-app/install
cub run deploy <space/unit>
cub run publish <space/unit>
cub run upgrade <space/unit>
cub run validate <space/unit>
```

### Initial flags

```bash
--assert
--record none|summary|full
--target <space/target>
--worker <space/worker>
--apply direct|argo|flux
--verbose
--open-gui
```

### CLI help sketch

```text
$ cub run --help

Run a bounded multi-step ConfigHub procedure.

Usage:
  cub run <procedure> [subject] [flags]
  cub run get <operation-id>
  cub run list
  cub run watch <operation-id>

Procedures:
  global-app/install  Materialize the global-app example and optionally apply it
  deploy     Apply or reconcile a ConfigHub-managed unit
  publish    Publish desired state for delegated reconciliation
  upgrade    Re-run an upgrade-oriented procedure and assertions
  validate   Run validation and assertion steps without deploying

Flags:
  --assert                    Show assertions and fail if any evaluated required assertion fails
  --record none|summary|full  Persist operation state to ConfigHub
  --target <space/target>     Target to use for remote execution
  --worker <space/worker>     Worker to use when required; otherwise the procedure may auto-resolve one
  --apply direct|argo|flux    Apply mode
  --verbose                   Show step-by-step progress
  --open-gui                  Print or open a GUI link when available
```

## MVP boundary

For MVP:

- only `cub run` emits `Operation` records
- existing commands such as `cub unit apply` and `cub function do` remain unchanged
- no `rerun`, `abort`, or step resume

This keeps the first implementation contained and avoids silently changing the contract of existing commands.

## Persistence defaults

### Core rule

For connected mutating procedures, `Operation` records should persist by default.

Local ephemeral mode should be explicit opt-out, not the implied norm.

### Recording modes

```text
--record=none     local or explicit opt-out; do not persist the Operation
--record=summary  persist operation state, step state, assertion state, bindings, refs, timestamps
--record=full     persist summary plus additional evidence refs
```

### Default behavior

If the user explicitly sets `--record`, that choice wins.

If the user does not set `--record`, MVP should behave like this:

- connected mutating procedures: `summary`
- local-only or unauthenticated procedures: `none`

### Latency requirement

Recording must not materially slow down the procedure.

For MVP:

- summary recording should be asynchronous and best-effort
- step execution must not block on a successful record write
- if a record write fails, the procedure continues and `cub` prints a warning
- `--record=full` may take longer, but that cost should be explicit and intentional

## `get`, `watch`, and `list`

For MVP, `cub run` should not require a long-lived server-side run engine.

Instead:

- `cub run` executes locally in the CLI
- `get` reads the stored `Operation` from ConfigHub
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

## `--assert` semantics

`--assert` should mean:

- evaluate assertions when evidence is available
- print assertion outcomes in the CLI
- return non-zero if any evaluated required assertion fails

For MVP:

- `pass` and `warn` do not fail the command
- `fail` does fail the command
- `pending` does not fail the command by itself

This matters for delegated flows.

A procedure may return with state `waiting` while assertions are still `pending`.
That should not be treated as a hard failure.

## Failure and partial completion

### Step failure

If a required step fails:

- the procedure stops by default
- the `Operation` is updated with the failed step
- earlier completed steps remain visible
- later steps remain `pending` or become `skipped`
- assertion state remains visible up to the point reached

### Waiting is not failure

If the command has completed its active work and is waiting on another system:

- the procedure state should be `waiting`
- the relevant step should stay `running`
- pending assertions should remain `pending`

That is common for delegated GitOps flows.

## Concrete examples

### Example 1: delegated ArgoCD flow

Current reality:

1. publish desired state
2. ArgoCD reconciles later
3. health checks happen after that

Without an `Operation`, the user sees command completion but still has to infer the rest.

With an `Operation`, ConfigHub can show:

- publish step done
- reconcile step waiting or done
- health assertions pending, pass, or fail

Failure-path example:

- publish step done
- reconcile step done
- health assertion fail
- operation state `failed`

### Example 2: live `global-app` install

Current reality:

1. create base spaces and base units
2. create environment spaces and clone/mutate units
3. resolve target and apply infra before app units

Without an `Operation`, the user has commands and scripts but no single operational record of the install procedure.

With an `Operation`, ConfigHub can show:

- whether preflight passed
- whether config materialization finished
- whether infra and app apply ran in the right order
- which assertions passed afterwards

## Worked CLI transcripts

### Delegated success then watch

```text
$ cub run publish argocd-guestbook --target dev/us-1 --apply argo --assert

Operation: op-01JARGO...
Procedure: publish
Subject: argocd-guestbook
Worker: dev/argocd-worker
Target: dev/us-1
Apply: argo

Step 1/5: preflight             DONE
Step 2/5: resolve target        DONE
Step 3/5: publish bundle        DONE
Step 4/5: wait for Argo sync    RUNNING
Step 5/5: post-deploy health    PENDING

Assertion: target reachable     PASS
Assertion: worker healthy       PASS
Assertion: argocd synced        PENDING
Assertion: guestbook healthy    PENDING

Procedure state: waiting
Done: no
```

### `global-app/install`

```text
$ cub run global-app/install --target dev/cluster-target --apply direct --assert

Operation: op-01JGLOBAL...
Procedure: global-app/install
Worker: dev/k8s-worker-1
Target: dev/cluster-target
Apply: direct

Step 1/4: preflight             DONE
Step 2/4: materialize-config    DONE
Step 3/4: apply                 DONE
Step 4/4: assert                RUNNING

Assertion: auth context         PASS
Assertion: target reachable     PASS
Assertion: env spaces created   PASS
Assertion: infra applied first  PASS
Assertion: app units healthy    PENDING

Procedure state: waiting
Done: no
```

Later:

```text
$ cub run watch op-01JARGO...

Operation: op-01JARGO...

Step 4/5: wait for Argo sync    DONE
Step 5/5: post-deploy health    DONE

Assertion: argocd synced        PASS
Assertion: guestbook healthy    PASS

Procedure state: done
Done: yes
```

### Delegated failure

```text
$ cub run watch op-01JARGO...

Operation: op-01JARGO...

Step 4/5: wait for Argo sync    DONE
Step 5/5: post-deploy health    DONE

Assertion: argocd synced        PASS
Assertion: guestbook healthy    FAIL

Procedure state: failed
Done: yes
```

## Why this is credible

We already have examples that expose the need.

### `incubator/cub-up/argocd-guestbook`

This example clearly separates direct apply from delegated apply and shows why `apply: argo` changes what “done” means.

### `incubator/global-app-layer/realistic-app`

This example already has a real multi-step shape spread across setup, target wiring, upgrade propagation, and verification.

### `gitops-import`

This example is the strongest existing evidence for `cub run`.

It requires a four-step manual sequence: create-cluster → install-argocd → setup-apps → install-worker. Each step must complete before the next can start. There is no shared state between steps. Success or failure is inferred from terminal output.

During e2e testing, the worker install step succeeded (pod running, 200 OK on stream) but silently failed to register targets because the org was at target quota. The worker logs showed the error, but `cub worker list` showed only "Disconnected" with no explanation. Diagnosing this required reading pod logs and cross-referencing quota limits — exactly the kind of multi-system status checking that `cub run` is designed to eliminate.

With `cub run`, this would be one operation with four steps and explicit assertions:

```text
Assertion: cluster reachable        PASS
Assertion: argocd healthy           PASS
Assertion: worker connected         FAIL — exceeded target quota
Assertion: targets registered       PENDING
```

The failure would be visible immediately instead of hidden behind a successful pod startup.

### Public examples: `global-app` and `helm-platform-components`

These show that the execution shape is not limited to recipe experiments.

## Success criteria

### User-facing success

- users can tell where a bounded procedure is
- users can tell whether it is done
- users can distinguish step completion from assertion completion
- users can review one shared operational record in ConfigHub

### Product success

- users spend less time guessing the next step
- users rerun fewer steps blindly
- users spend less time checking multiple systems for status
- interrupted work is easier to resume or hand off
- example scripts get thinner over time

## Open questions

1. What is the right ConfigHub backing type for `Operation` records?
2. What is the minimum assertion set for `global-app/install`, `deploy`, and `publish`?
3. When should `cub run` stop waiting automatically versus return `waiting`?
4. When, if ever, should procedure profiles become declarative rather than hardcoded?

## If the CLI version proves useful

If the CLI version proves useful, the next step should not be to keep growing CLI-only behavior.

The next step should be to add first-class operational support in ConfigHub itself, with:

- an `Operation` record for one procedure
- ordered phases or steps
- typed assertions with evidence
- explicit bindings to subject, worker, target, apply mode, and bundle
- recorded state transitions over time
- one shared model for CLI, GUI, AI, and API

In that model, `cub run` becomes the OSS CLI over ConfigHub's native operational data, not the place where the concept mainly lives.
