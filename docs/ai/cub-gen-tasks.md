# cub-gen Task Skill (For AI Agents Using cub-gen)

> **Audience:** AI agents (Claude, Codex, etc.) helping a human operator
> investigate, govern, or trace changes through a GitOps source repo.
>
> **Difference from `cub-gen-skill.md`:** That doc is the capability-question
> profile ("can cub-gen do X?"). This doc is task-oriented for *using*
> cub-gen on a real source repo.

## Core mental model

`cub-gen` is the **repo-side traceability and governed-change CLI**. It starts
from DRY source files (the things humans edit), maps them to WET rendered
manifests (the things controllers deploy), and answers four questions:

1. **What generators is this repo using?** — Helm, Score, Spring Boot, etc.
2. **Where does this deployed field come from?** — DRY file/path/owner
3. **Where do I edit this safely?** — inverse-edit hint with confidence score
4. **What evidence exists for this change?** — provenance bundle, attestation

cub-gen never deploys to a cluster. It runs locally against a source repo and
optionally talks to ConfigHub for governed workflows.

When the local render target is the same repo path, omit the second path
argument. Use the explicit two-path form only when you need different source
and render target paths.

## Task → command map

When the operator asks... | Run this | What you get
---|---|---
"What generators does this repo use?" | `./cub-gen detect --repo $REPO --pretty` | Detected generator profiles
"What generators are supported?" | `./cub-gen generators --markdown --details` | Full generator catalog
"Render and show provenance" | `./cub-gen gitops import --space my-space $REPO --json` | Rendered manifests + field origin map
"Build a provenance bundle" | `./cub-gen publish --space my-space $REPO --pretty` | Bundle JSON ready for verify/attest
"Verify a bundle" | `./cub-gen publish ... \| ./cub-gen verify --in -` | Pass/fail with reasons
"Sign a bundle" | `./cub-gen publish ... \| ./cub-gen attest --in - --verifier ci-bot` | Signed attestation
"Preview a change" | `./cub-gen change preview --space my-space $REPO` | Diff + provenance + confidence
"Where do I edit this field?" | `./cub-gen change explain --wet-path "<path>" $REPO` | DRY file/path/owner
"Run a governed change" | `./cub-gen change run --mode local --space my-space $REPO` | Local change report (or connected to ConfigHub)
"Verify the connected smoke path" | `cub auth login && ./examples/demo/run-connected-smoke.sh` | ConfigHub auth/context + flagship connected wrappers
"Use the deep bridge API path" | `./cub-gen bridge ingest --in bundle.json --base-url <url>` | Ingest receipt
"Check deep decision status" | `./cub-gen bridge decision query --change-id <id> --base-url <url>` | ALLOW / ESCALATE / BLOCK

## First commands for any source repo

```bash
./cub-gen --help                                  # verify cub-gen is available
./cub-gen detect --repo $REPO --pretty            # what generators does this repo use?
./cub-gen gitops discover --space my-space $REPO  # what artifacts can be rendered?
```

That gives you: tool version, generator detection, and discoverable artifacts —
enough to answer most "what is this repo doing?" questions.

## Detailed task flows

### Task: Trace a deployed field back to source

```bash
# 1. Build a publish bundle and inspect field origin
./cub-gen publish --space my-space $REPO --pretty | jq '.provenance[0].field_origin_map'

# 2. Or query a specific WET path directly
./cub-gen change explain --space my-space \
  --wet-path "Deployment/spec/template/spec/containers[0]/image" \
  $REPO
```

The output includes:
- `dry_path` — the source field that produced it
- `source_path` — the file (e.g., `values.yaml`)
- `transform` — which generator did the mapping (e.g., `helm-template`)
- `owner` — which team owns the source field
- `edit_hint` — plain-English instruction
- `confidence` — score 0–1 (use thresholds: ≥0.90 normal, 0.75–0.89 preview first, <0.75 escalate)

### Task: Validate a proposed change

```bash
# 1. Preview the change locally
./cub-gen change preview --space my-space $REPO --json

# 2. Run it (still local, just produces a change report)
./cub-gen change run --mode local --space my-space $REPO --json

# 3. (Connected, advanced) Send for governed decision
./cub-gen change run --mode connected --base-url <confighub-url> \
  --space my-space $REPO --json
```

In the advanced connected path, ConfigHub returns a decision: ALLOW / ESCALATE / BLOCK.

### Task: Build an evidence bundle for a release

```bash
./cub-gen publish --space my-space $REPO \
  | ./cub-gen verify --in - \
  | ./cub-gen attest --in - --verifier ci-bot \
  > release-evidence.json
```

The chained pipeline gives you: provenance → verified → signed. Save the
final attestation alongside the release artifact.

### Task: Investigate a generator detection failure

```bash
# 1. See what was detected
./cub-gen detect --repo $REPO --pretty

# 2. See what's supported
./cub-gen generators --markdown --details

# 3. If the repo uses something supported but cub-gen missed it,
#    check the discovery output for hints
./cub-gen gitops discover --space my-space $REPO --json | jq .
```

If a generator is not in the supported list, that's a feature gap — file an
issue rather than guessing.

## Output interpretation

### Confidence scores

cub-gen reports confidence on every field-origin mapping:

| Score | Routing |
|-------|---------|
| ≥ 0.90 | Normal app/team edit flow |
| 0.75 – 0.89 | Run `change preview` and `change explain` before merge |
| < 0.75 | Escalate for platform review |

Always surface confidence to the operator when reporting field origins.

### Field origin shape

```json
{
  "dry_path": "values.image.tag",
  "wet_path": "Deployment/spec/template/spec/containers[0]/image",
  "source_path": "values.yaml",
  "transform": "helm-template",
  "confidence": 0.86
}
```

### Inverse edit pointer shape

```json
{
  "wet_path": "Deployment/spec/template/spec/containers[0]/image",
  "dry_path": "values.image.tag",
  "owner": "app-team",
  "edit_hint": "Edit chart values file and keep chart template unchanged.",
  "confidence": 0.86
}
```

The `edit_hint` is plain-English and safe to relay verbatim to the operator.

### Generator profiles supported today

Run `./cub-gen generators --markdown --details` for the live list. As of v0.2:
Helm, Score.dev, Spring Boot, Backstage, No-Config-Platform, Ops Workflow,
C3 Agent, Swamp.

## When NOT to use cub-gen

cub-gen is source-side and read-only against the cluster. It cannot:

| Task | Use this instead |
|------|------------------|
| Deploy to a cluster | Flux, ArgoCD, `kubectl apply` |
| Read live cluster state | `cub-scout` |
| Rollback a release | Argo UI, Flux suspend, Helm rollback |
| Run a Kyverno/OPA policy | ConfigHub policy engine, OPA, Kyverno directly |
| Edit source files for the user | Operators must edit themselves; cub-gen tells them where |
| Render a generator that isn't in the supported list | File a feature request |

## Verification rule

Before claiming a command exists, verify with `--help`:

```bash
./cub-gen --help
./cub-gen <command> --help
./cub-gen generators --json   # for current generator support
```

Command surface and generator catalog evolve. Use `--help` and `generators
--json` as source of truth.

## Local vs connected mode

```bash
./cub-gen change run --mode local      # local report, no backend
./examples/demo/run-connected-smoke.sh # connected smoke lane
./cub-gen change run --mode connected --base-url <url>  # ConfigHub decision
```

Start with the connected smoke lane after `cub auth login`. The direct
`change run --mode connected` path is deeper and requires a ConfigHub base URL
plus the backend bridge endpoints. If the user does not have ConfigHub set up,
all the source-side and provenance work still works in local mode.

## Variants and overlays

Current status: partial support.

- A single `gitops import` / `publish` invocation works on one repo path pair.
- If the repo already contains generator-native overlay files, `cub-gen` can include them in provenance and dry input reporting.
- For example, Spring Boot can emit both base and overlay field-origin entries for `server.port`, while `change explain` points to the profile-specific edit path:

```bash
./cub-gen gitops import --space my-space --json ./examples/springboot-paas \
  | jq '.provenance[0].field_origin_map[] | select(.dry_path=="server.port")'

./cub-gen change explain --space my-space \
  --wet-path "Deployment/spec/template/spec/containers[0]/ports[0]/containerPort" \
  ./examples/springboot-paas
```

- There is no current CLI for repeated `--values`, `--overlay`, `--variant`, or globbed fan-out that emits one bundle per environment in a single command.
- If the user asks for per-environment bundle fan-out, describe it as a current product gap instead of inventing a flag.

## JSON-first for automation

Most cub-gen commands support `--json` and `--pretty` flags. Prefer JSON
output when the result will be parsed:

```bash
./cub-gen detect --repo $REPO --pretty
./cub-gen generators --json --details
./cub-gen gitops import --space my-space $REPO $REPO --json
./cub-gen publish --space my-space $REPO $REPO --pretty
./cub-gen change preview --space my-space $REPO $REPO --json
./cub-gen change explain --space my-space --wet-path "<path>" $REPO $REPO --json
```

The contract is documented in `docs/contracts/change-cli-v1.md`.

## Honest limitations to report to operators

If asked about something cub-gen cannot do today, do not invent. Be honest:

- **Generator coverage is finite** — only generators in `./cub-gen generators` are supported
- **Confidence is a heuristic** — not a guarantee of correctness; threshold-based routing exists for a reason
- **No cluster reach** — cub-gen never reads or writes a cluster; that's `cub-scout`'s job
- **Connected smoke and deep connected paths require ConfigHub** — the smoke lane checks auth/context; bridge workflows also depend on backend endpoints
- **Inverse-edit hints are advisory** — they tell you where to edit, they don't perform the edit

When a capability is missing, offer to file an issue at
<https://github.com/confighub/cub-gen/issues>.

## How cub-gen relates to cub-scout

| Question | Tool |
|----------|------|
| What's running in my cluster? | `cub-scout` |
| Where did this deployed field come from? | `cub-gen` |
| Who owns this resource? | `cub-scout` (controller-level) and `cub-gen` (field-level) |
| Why is this resource broken? | `cub-scout explain` |
| Where do I edit this safely? | `cub-gen change explain` |
| Is this change governed? | `cub-gen change run --mode connected` (with ConfigHub) |

Use them together: `cub-scout` answers "what's broken in the cluster" and
`cub-gen` answers "where in the source repo do I fix it".
