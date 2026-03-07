# Demo Illustration Pack

**Purpose:** Ready-to-use visuals for meetings, demos, and docs.

Qualification rule:
Use `Agentic GitOps` only when an active inner reconciliation loop (`WET -> LIVE`) exists via Flux/Argo (or equivalent reconciler). Without that loop, classify the flow as `governed config automation`.

## 1. Layered responsibility diagram

```mermaid
flowchart TB
  U["Users and Agents"] --> C["ConfigHub API + Governance"]
  C --> P["Publish layer\nOCI/Git"]
  P --> R["Runtime reconciliation\nFlux/Argo"]
  R --> L["LIVE clusters"]
  L --> O["Observation + Evidence\ncub-scout"]
  O --> C
```

## 2. Decision split (review vs deploy)

```mermaid
flowchart LR
  A["Change proposal"] --> B["Merge review"]
  B --> C{"Deploy decision"}
  C -- "ALLOW" --> D["Execute tokened apply"]
  C -- "ESCALATE" --> E["Extra approval"]
  C -- "BLOCK" --> F["Stop"]
  D --> G["Verify + attest"]
```

## 3. DRY/WET/LIVE lifecycle

```mermaid
sequenceDiagram
  participant Dev as Developer or Agent
  participant Gen as Generator
  participant CH as ConfigHub
  participant GitOps as Flux/Argo
  participant Live as LIVE Cluster
  participant Obs as cub-scout

  Dev->>Gen: DRY input
  Gen->>CH: WET + contract + provenance
  CH->>GitOps: Publish (OCI/Git)
  GitOps->>Live: Reconcile
  Obs->>CH: Evidence + drift
  CH->>Dev: Decision + attestation result
```

## 4. Fast demo overlay script (talk track)

1. Show runtime unchanged (Flux/Argo still reconcile).
2. Show change proposal and decision split.
3. Show one enforcement event (BLOCK or ESCALATE).
4. Show successful ALLOW path with attestation.
5. Show mutation ledger record.

## 5. Reusable slide captions

1. `No controller replacement: runtime stays Flux/Argo.`
2. `From file edits to governed platform API decisions.`
3. `Proof-first operations: verify before success claims.`
4. `One change identity across intent, execution, and outcome.`
