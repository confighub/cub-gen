# Change API Contract v1 (Draft)

Status: Draft
Version: `1.0.0-draft`

This contract defines the HTTP API surface that mirrors `cub-gen change preview|run|explain`.

Goal: CI systems, agents, and UIs can consume one JSON contract without shelling across multiple files.

## Endpoints

### 1) Create or execute a change

`POST /v1/changes`

Request fields:

- `action`: `preview` or `run`
- `mode`: `local` or `connected` (required for `action=run`)
- `input`: target/render target + labels (`space`, `ref`, `where_resource`)
- `connected`: optional backend connection fields (`base_url`, `token`, endpoint overrides)

Response:

- `change` (`change_id`, `bundle_digest`, `attestation_digest`)
- `edit_recommendation` (owner, paths, hint, confidence)
- `verification` (bundle/attestation validity)
- `decision` and `promotion_ready` when `action=run`
- `artifacts` references for audit/debug

Schema:

- `docs/contracts/schemas/change-request.v1.schema.json`

### 2) Get change status

`GET /v1/changes/{change_id}`

Response:

- `change`
- `decision`
- `verification`
- `artifacts`

Schema:

- `docs/contracts/schemas/change-decision.v1.schema.json`

### 3) Explain what to edit

`GET /v1/changes/{change_id}/explanations`

Query params:

- `wet_path` (optional)
- `dry_path` (optional)
- `owner` (optional)

Response:

- `query` (filters + match count)
- `explanation` (owner, source path, dry/wet path, edit hint, confidence)

Schema:

- `docs/contracts/schemas/change-explanation.v1.schema.json`

## Example requests

### `POST /v1/changes` (preview)

```json
{
  "action": "preview",
  "input": {
    "target_slug": "./examples/scoredev-paas",
    "render_target_slug": "./examples/scoredev-paas",
    "space": "platform",
    "ref": "HEAD"
  }
}
```

### `POST /v1/changes` (run connected)

```json
{
  "action": "run",
  "mode": "connected",
  "input": {
    "target_slug": "./examples/helm-paas",
    "render_target_slug": "./examples/helm-paas",
    "space": "platform"
  },
  "connected": {
    "base_url": "https://confighub.example",
    "token": "${CONFIGHUB_TOKEN}"
  }
}
```

### `GET /v1/changes/{change_id}/explanations?wet_path=...`

```json
{
  "change": {
    "change_id": "chg_01J...",
    "bundle_digest": "sha256:...",
    "attestation_digest": "sha256:..."
  },
  "query": {
    "wet_path_filter": "Deployment/spec/template/spec/containers[name=main]/image",
    "match_count": 1
  },
  "explanation": {
    "owner": "app-team",
    "wet_path": "Deployment/spec/template/spec/containers[name=main]/image",
    "dry_path": "containers.main.image",
    "edit_hint": "Edit the Score container image in score.yaml.",
    "confidence": 0.94,
    "source_path": "score.yaml",
    "source_transform": "score-container-to-deployment"
  }
}
```

## Error contract

HTTP status guidance:

- `200`: success
- `400`: invalid request payload or missing required filters
- `401`: auth missing/invalid for connected mode
- `404`: `change_id` not found
- `409`: policy conflict or non-terminal decision constraints
- `422`: contract/triple validation failure
- `503`: backend dependency unavailable

Error body shape:

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "change run requires mode=local|connected",
    "details": {
      "field": "mode"
    }
  }
}
```

## Mapping to CLI

- `POST /v1/changes {action:"preview"}` ↔ `cub-gen change preview ...`
- `POST /v1/changes {action:"run"}` ↔ `cub-gen change run ...`
- `GET /v1/changes/{change_id}/explanations` ↔ `cub-gen change explain ...`

Compatibility adapter in this repo:

- `examples/demo/change-api-adapter.sh`

## See also

- `docs/contracts/change-cli-v1.md`
- `docs/contracts/decision-and-attestation-state.md`
