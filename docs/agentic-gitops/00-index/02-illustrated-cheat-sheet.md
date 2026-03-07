# Illustrated Cheat Sheet

**Purpose:** Fast visual reference for the operating model and demo narrative.

Qualification rule:
Use `Agentic GitOps` only when an active inner reconciliation loop (`WET -> LIVE`) exists via Flux/Argo (or equivalent reconciler). Without that loop, classify the flow as `governed config automation`.

## 1. Three-loop model

```mermaid
flowchart LR
  A["DRY intent"] --> B["Generator\n(DRY -> WET)"]
  B --> C["WET intended state"]
  C --> D["Flux/Argo reconcile\n(WET -> LIVE)"]
  D --> E["LIVE runtime"]
  E --> F["Evidence + verification"]
  F --> G["Decision + attestation"]
  G --> C
```

## 2. Contract triple

```mermaid
flowchart TB
  GC["GeneratorContract\nSigned\nDeterministic hash"]
  PR["ProvenanceRecord\ninput_hash\ntoolchain_version\npolicy_version\nrun_id\nartifacts"]
  IP["InverseTransformPlan\nOwnershipMap scope\nReplay check\nDecision gate"]

  GC --> PR
  PR --> IP
  IP --> WB["PR/MR write-back only"]
  IP --> DG["ALLOW | ESCALATE | BLOCK"]
  DG --> AT["Attestation on ALLOW"]
```

## 3. Enforcement outcomes

```mermaid
flowchart TD
  A["Proposed change"] --> B{"Ownership scope valid?"}
  B -- "No" --> B1["BLOCK"]
  B -- "Yes" --> C{"Replay hash match?"}
  C -- "No" --> C1["ESCALATE"]
  C -- "Yes" --> D{"Policy decision"}
  D -- "BLOCK" --> D1["Stop"]
  D -- "ESCALATE" --> D2["Manual approval"]
  D -- "ALLOW" --> E["Execute"]
  E --> F{"Verification pass?"}
  F -- "No" --> F1["Read-only evidence mode"]
  F -- "Yes" --> G["Attest + ledger append"]
```

## 4. 2x4 demo map

```mermaid
flowchart LR
  subgraph P["Platform/App track"]
    H["1. Helm PaaS"] --> S["2. Score.dev"] --> SB["3. Spring Boot"] --> A["4. Ably config"]
  end

  subgraph W["AI work platform track"]
    J["5. Jesper AI cloud"] --> SW["6. Swamp project"] --> CA["7. ConfigHub Actions"] --> O["8. Ops workflow"]
  end
```

## 5. One-line boundary

`Flux/Argo reconcile. ConfigHub decides. Git records.`
